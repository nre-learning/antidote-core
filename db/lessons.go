package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
			log.Errorf("Failed to insert lesson '%s' into the database: %v", lesson.Slug, err)
			continue
		}

		log.Infof("Successfully imported lesson '%s'  with %d endpoints.", lesson.Slug, len(lesson.Endpoints))

		retLds[lesson.Id] = &lesson
	}

	// TODO(mierdin): need to load into memory first so you can do inter-lesson references (like prereqs) first,
	// and THEN insert into DB.

	if len(fileList) == len(retLds) {
		log.Infof("Imported %d lesson definitions.", len(retLds))
		return nil
	} else {
		log.Warnf("Imported %d lesson definitions with errors.", len(retLds))
		return errors.New("Not all lesson definitions were imported")
	}

}

// TODO(mierdin): These need to be named more appropriately to be more consistent, and namespaced to lessons,
// as other resources will have their own.
var (
	BasicValidationError          error = errors.New("a")
	TierMismatchError             error = errors.New("a")
	InsufficientPresentationError error = errors.New("a")
	ProhibitedImageTagError       error = errors.New("a")
	InvalidConfigurationType      error = errors.New("a")
	MissingConfigurationFile      error = errors.New("a")
	DuplicatePresentationError    error = errors.New("a")
	MissingPresentationPort       error = errors.New("a")
	BadConnectionError            error = errors.New("a")
	UnsupportedGuideTypeError     error = errors.New("a")
	MissingLessonGuide            error = errors.New("a")
)

// validateLesson validates a single lesson, returning a simple error if the lesson fails
// to validate.
func validateLesson(syringeConfig *config.SyringeConfig, lesson *models.Lesson) error {

	file := lesson.LessonFile

	// TODO(mierdin) Work to move as much validation as possible out of validateLesson and into
	// jsonschema.
	if !lesson.JSValidate() {
		log.Errorf("Basic schema validation failed on %s - see log for errors.", file)
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

		// Perform configuration-related checks, if relevant
		if ep.ConfigurationType != "" {

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
		}

		// Ensure each presentation name is unique for each endpoint
		seenPresentations := map[string]*models.LessonPresentation{}
		for n := range ep.Presentations {
			if _, ok := seenPresentations[ep.Presentations[n].Name]; ok {
				log.Errorf("Failed to import %s: - Presentation %s appears more than once for an endpoint", file, ep.Presentations[n].Name)
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
			log.Errorf("Failed to import %s: - Connection %s refers to nonexistent entity", file, connection.A)
			return BadConnectionError
		}

		if !entityInLabDef(connection.B, lesson) {
			log.Errorf("Failed to import %s: - Connection %s refers to nonexistent entity", file, connection.B)
			return BadConnectionError
		}
	}

	// TODO(mierdin): Check to make sure referenced collection exists

	// TODO(mierdin): Make sure lesson ID, lesson name, stage ID, stage name, and endpoint name are all unique.
	// I don't believe this is possible to do in JSONSchema, so we'll need to add a check for it here.

	// TODO(mierdin): Need to validate that each name is unique across endpoints

	// TODO(mierdin): Need to run checks to see that files are located where they need to be. Things like
	// configs, and lesson guides

	// Iterate over stages, and retrieve lesson guide content
	for l := range lesson.Stages {
		s := lesson.Stages[l]

		guideFileMap := map[string]string{
			"markdown": ".md",
			"jupyter":  ".ipynb",
		}

		if _, ok := guideFileMap[s.GuideType]; !ok {
			log.Errorf("Failed to import %s: - stage references an unsupported guide type", file)
			return UnsupportedGuideTypeError
		}

		fileName := fmt.Sprintf("%s/stage%d/guide%s", filepath.Dir(file), s.Id, guideFileMap[s.GuideType])
		contents, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Errorf("Encountered problem reading lesson guide: %s", err)
			return MissingLessonGuide
		}
		lesson.Stages[l].LabGuide = string(contents)
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
