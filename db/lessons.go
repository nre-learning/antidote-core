package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pg "github.com/go-pg/pg"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	config "github.com/nre-learning/syringe/config"
	models "github.com/nre-learning/syringe/db/models"
)

// InsertLessons takes a slides of lesson definitions, and inserts them into the database.
// It is a really good idea to only use slices returned from ReadLessons() as input for this function.
func (a *AntidoteDB) InsertLessons(lessons []*models.Lesson) error {

	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// Rollback tx on error.
	defer tx.Rollback()

	for i := range lessons {
		err := tx.Insert(lessons[i])
		if err != nil {
			log.Errorf("Failed to insert lesson '%s' into the database: %v", lessons[i].Slug, err)
			return err
		}
	}

	return tx.Commit()
}

// ReadLessons reads lesson definitions from the filesystem, validates them, and returns them
// in a slice.
func (a *AntidoteDB) ReadLessons() ([]*models.Lesson, error) {

	// Get lesson definitions
	fileList := []string{}
	lessonDir := fmt.Sprintf("%s/lessons", a.SyringeConfig.CurriculumDir)
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
		return nil, err
	}

	retLds := []*models.Lesson{}

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

		err = validateLesson(a.SyringeConfig, &lesson)
		if err != nil {
			log.Errorf("Lesson '%s' failed to validate", lesson.Slug)
			continue
		}

		// Insert stage at zero-index so we can use slice indexes to refer to each stage without jumping through hoops
		// or making the user use 0 as a stage ID
		// TODO(mierdin): Giant code smell. Kill it.
		// lesson.Stages = append([]*models.LessonStage{{Id: 0}}, lesson.Stages...)

		log.Infof("Successfully imported lesson '%s'  with %d endpoints.", lesson.Slug, len(lesson.Endpoints))

		retLds = append(retLds, &lesson)
	}

	// TODO(mierdin): need to load into memory first so you can do inter-lesson references (like prereqs) first,
	// and THEN insert into DB.

	if len(fileList) == len(retLds) {
		log.Infof("Imported %d lesson definitions.", len(retLds))
		return retLds, nil
	} else {
		log.Warnf("Imported %d lesson definitions with errors.", len(retLds))
		return retLds, errors.New("Not all lesson definitions were imported")
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
	MissingCheckerScript          error = errors.New("a")
)

// validateLesson validates a single lesson, returning a simple error if the lesson fails
// to validate.
func validateLesson(syringeConfig *config.SyringeConfig, lesson *models.Lesson) error {

	/* TODO(mierdin): Need to add these checks:

	- Make sure referenced collection exists
	- Make sure lesson ID, lesson name, stage ID, stage name, and endpoint name are all unique.
	  I don't believe this is possible to do in JSONSchema, so we'll need to add a check for it here.
	*/

	file := lesson.LessonFile

	// Most of the validation heavy lifting should be done via JSON schema as much as possible.
	// This should be run first, and then only checks that can't be done with JSONschema will follow.
	if !lesson.JSValidate() {
		log.Errorf("Basic schema validation failed on %s - see log for errors.", file)
		return BasicValidationError
	}

	// Endpoint-specific checks
	for i := range lesson.Endpoints {
		ep := lesson.Endpoints[i]

		// Must EITHER provide additionalPorts, or Presentations. Endpoints without either are invalid.
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
		seenPresentations := map[string]string{}
		for n := range ep.Presentations {
			if _, ok := seenPresentations[ep.Presentations[n].Name]; ok {
				log.Errorf("Failed to import %s: - Presentation %s appears more than once for an endpoint", file, ep.Presentations[n].Name)
				return DuplicatePresentationError
			}
			seenPresentations[ep.Presentations[n].Name] = ep.Presentations[n].Name
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

		// Ensure the necessary checker script is present for all stages
		for s := range lesson.Stages {
			for i := range lesson.Stages[s].Objectives {
				fileName := fmt.Sprintf("%s/stage%d/checkers/%d.py", filepath.Dir(file), lesson.Stages[s].Id, lesson.Stages[s].Objectives[i].Id)
				_, err := ioutil.ReadFile(fileName)
				if err != nil {
					log.Errorf("Checker script %s was not found.", fileName)
					return MissingCheckerScript
				}
			}
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

// ListLessons retrieves lessons present in the database
func (a *AntidoteDB) ListLessons() ([]*models.Lesson, error) {

	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	var lessons []*models.Lesson
	err := db.Model(&lessons).Select()
	if err != nil {
		return nil, err
	}

	return lessons, nil
}

// GetLesson retrieves a specific lesson via slug from the database
func (a *AntidoteDB) GetLesson(slug string) (*models.Lesson, error) {

	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	var lesson models.Lesson
	_, err := db.QueryOne(&lesson, `SELECT * FROM lessons WHERE slug = ?`, slug)
	if err != nil {
		return nil, err
	}

	return &lesson, nil
}
