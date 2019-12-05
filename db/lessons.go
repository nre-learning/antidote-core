package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	pg "github.com/go-pg/pg"

	config "github.com/nre-learning/syringe/config"
	models "github.com/nre-learning/syringe/db/models"
)

// ImportLessons is responsible for the entire workflow of importing lesson content, from
// the filesystem into the database.
func (a *AntidoteDB) ImportLessons(syringeConfig *config.SyringeConfig) error {

	// Connect to Postgres
	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	// Get lesson definitions
	fileList := []string{}
	lessonDir := fmt.Sprintf("%s/lessons", syringeConfig.CurriculumDir)
	log.Debugf("Searching %s for lesson definitions", lessonDir)
	err := filepath.Walk(lessonDir, func(path string, f os.FileInfo, err error) error {
		syringeFileLocation := fmt.Sprintf("%s/lesson.meta.yaml", path)
		if _, err := os.Stat(syringeFileLocation); err == nil {
			log.Debugf("Found lesson definition at: %s", syringeFileLocation)
			fileList = append(fileList, syringeFileLocation)
		}
		return nil
	})
	if err != nil {
		return err
	}

	retLds := map[int32]*models.Lesson{}

	for f := range fileList {

		file := fileList[f]

		log.Infof("Importing lesson definition at: %s", file)

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue
		}

		var lesson models.Lesson
		err = yaml.Unmarshal([]byte(yamlDef), &lesson)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
		}
		lesson.LessonFile = file
		lesson.LessonDir = filepath.Dir(file)

		err = validateLesson(syringeConfig, &lesson)
		if err != nil {
			log.Errorf("Lesson %d failed to validate", lesson.Id)
			continue
		}

		// Insert stage at zero-index so we can use slice indexes to refer to each stage without jumping through hoops
		// or making the user use 0 as a stage ID
		// TODO(mierdin): Giant code smell. Kill it.
		lesson.Stages = append([]*models.LessonStage{{Id: 0}}, lesson.Stages...)

		err = db.Insert(&lesson)
		if err != nil {
			log.Errorf("Failed to insert lesson '%s' into the database: %v", lesson.Id, err)
			continue
		}

		log.Infof("Successfully imported lesson '%s'  with %d endpoints.", lesson.Slug, len(lesson.Endpoints))

		retLds[lesson.Id] = &lesson
	}

	if len(fileList) == len(retLds) {
		log.Infof("Imported %d lesson definitions.", len(retLds))
		return nil
	} else {
		log.Warnf("Imported %d lesson definitions with errors.", len(retLds))
		return errors.New("Not all lesson definitions were imported")
	}

}

var (
	BasicValidationError          error
	TierMismatchError             error
	InsufficientPresentationError error
	ProhibitedImageTagError       error
	InvalidConfigurationType      error
	MissingConfigurationFile      error
	DuplicatePresentationError    error
	MissingPresentationPort       error
)

// validateLesson validates a single lesson, returning a simple error if the lesson fails
// to validate.
func validateLesson(syringeConfig *config.SyringeConfig, lesson *models.Lesson) error {

	// TODO(mierdin): In the future, you should consider putting unique error messages for
	// each violation. This will make this function more testable.
	// fail := errors.New("failed to validate lesson definition")
	file := lesson.LessonFile

	if !models.JSValidate(lesson) {
		// TODO(mierdin) fix err
		// log.Errorf("Basic validation failed on %s: %s", file, err)
		log.Errorf("Basic schema validation failed on %s: %s", file)
		return BasicValidationError
	}

	// More advanced validation
	if syringeConfig.Tier == "prod" {
		if lesson.Tier != "prod" {
			log.Errorf("Skipping %s: lower tier than configured", file)
			return TierMismatchError
		}
	} else if syringeConfig.Tier == "ptr" {
		if lesson.Tier != "prod" && lesson.Tier != "ptr" {
			log.Errorf("Skipping %s: lower tier than configured", file)
			return TierMismatchError
		}
	}

	// Ensure each device in the definition has a corresponding config for each stage
	for i := range lesson.Endpoints {

		ep := lesson.Endpoints[i]

		if len(ep.Presentations) == 0 && len(ep.AdditionalPorts) == 0 {
			log.Error("No presentations configured, and no additionalPorts specified")
			return InsufficientPresentationError
		}

		if strings.Contains(ep.Image, ":") {
			log.Error("Tags are not allowed in endpoint image refs")
			return ProhibitedImageTagError
		}

		// If this endpoint has no configurationtype, there's nothing else to check, and we can
		// continue. NOTE that ONLY configuration validation logic is to follow this.
		if ep.ConfigurationType == "" {
			continue
		}

		// List of supported configuration options. Note tht this is a list of regex match statements,
		// not literal string matches, so we can get include the NAPALM driver in the string.
		supportedConfigurationOptions := []string{
			"python",
			"ansible",
			"napalm-.*$",
		}

		matchedOne := false
		for o := range supportedConfigurationOptions {
			matched, err := regexp.Match(supportedConfigurationOptions[o], []byte(ep.ConfigurationType))
			if err != nil {
				log.Error("Unable to determine configurationType")
				return InvalidConfigurationType
			}
			if matched {
				matchedOne = true
				break
			}
		}
		if !matchedOne {
			log.Error("Unsupported configurationType")
			return InvalidConfigurationType
		}

		fileMap := map[string]string{
			"python":  ".py",
			"ansible": ".yml",
			"napalm":  ".txt",
		}
		// All NAPALM drivers will use the same file extension so we only need everything before the hyphen to
		// make this work
		fileExt := fileMap[strings.Split(ep.ConfigurationType, "-")[0]]

		// Ensure the necessary config file is present for all stages
		for s := range lesson.Stages {
			fileName := fmt.Sprintf("%s/stage%d/configs/%s%s", filepath.Dir(file), lesson.Stages[s].Id, ep.Name, fileExt)
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Configuration script %s was not found.", fileName)
				return MissingConfigurationFile
			}
		}

		// Ensure each presentation name is unique for each endpoint
		seenPresentations := map[string]*models.LessonPresentation{}
		for n := range ep.Presentations {
			if _, ok := seenPresentations[ep.Presentations[n].Name]; ok {
				log.Errorf("Failed to import %s: - Presentation %s appears more than once for an endpoint", file, ep.Presentations[n].Name)))
				return DuplicatePresentationError
			}

			if ep.Presentations[n].Port == 0 {
				log.Error("All presentations must specify a port")
				return MissingPresentationPort
			}

			seenPresentations[ep.Presentations[n].Name] = ep.Presentations[n]
		}
	}

	// Ensure all connections are referring to endpoints that are actually present in the definition
	for c := range lesson.Connections {
		connection := lesson.Connections[c]

		if !entityInLabDef(connection.A, lesson) {
			log.Errorf("Failed to import %s: - Connection %s refers to nonexistent entity", file, connection.A)))
			return fail
		}

		if !entityInLabDef(connection.B, lesson) {
			log.Errorf("Failed to import %s: %s - Connection %s refers to nonexistent entity", file, connection.B)))
			return fail
		}
	}

	// TODO(mierdin): Check to make sure referenced collection exists

	// TODO(mierdin): Make sure lesson ID, lesson name, stage ID and stage name are unique.
	// If you try to read a value in, make sure it doesn't exist. If it does, error out

	// TODO(mierdin): Need to validate that each name is unique across endpoints

	// TODO(mierdin): Need to run checks to see that files are located where they need to be. Things like
	// configs, and lesson guides

	// Iterate over stages, and retrieve lesson guide content
	for l := range lesson.Stages {
		s := lesson.Stages[l]

		if s.VerifyCompleteness == true {
			fileName := fmt.Sprintf("%s/stage%d/verify.py", filepath.Dir(file), s.Id)
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Stage specified VerifyCompleteness but no verify.py script was found: %s", err)
				return fail
			}
		}

		// Validate presence of jupyter notebook in expected location
		if s.JupyterLabGuide == true {
			fileName := fmt.Sprintf("%s/stage%d/notebook.ipynb", filepath.Dir(file), s.Id)
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Stage specified a jupyter notebook lesson guide, but the file was not found: %s", err)
				return fail
			}
		} else {
			fileName := fmt.Sprintf("%s/stage%d/guide.md", filepath.Dir(file), s.Id)
			contents, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Encountered problem reading lesson guide: %s", err)
				return fail
			}
			lesson.Stages[l].LabGuide = string(contents)
		}

		if s.VerifyCompleteness == true && s.VerifyObjective == "" {
			log.Error("Must provide a VerifyObjective for stages with VerifyCompleteness set to true")
			return fail
		}

	}

	return nil
}

// entityInLabDef is a helper function to ensure that a device is found by name in a lab definition
func entityInLabDef(entityName string, ld *models.Lesson) bool {

	for i := range ld.Endpoints {
		if entityName == ld.Endpoints[i].Name {
			return true
		}
	}
	return false
}
