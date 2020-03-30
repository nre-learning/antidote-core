package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	models "github.com/nre-learning/antidote-core/db/models"
)

// ReadLessons reads lesson definitions from the filesystem, validates them, and returns them
// in a slice.
func ReadLessons(curriculumDir string) ([]*models.Lesson, error) {

	// Get lesson definitions
	fileList := []string{}
	lessonDir := fmt.Sprintf("%s/lessons", curriculumDir)
	log.Debugf("Searching %s for lesson definitions", lessonDir)
	err := filepath.Walk(lessonDir, func(path string, f os.FileInfo, err error) error {
		lessonDefFile := fmt.Sprintf("%s/lesson.meta.yaml", path)
		if _, err := os.Stat(lessonDefFile); err == nil {
			log.Debugf("Found lesson definition at: %s", lessonDefFile)
			fileList = append(fileList, lessonDefFile)
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

		err = validateLesson(&lesson)
		if err != nil {
			log.Errorf("Lesson '%s' failed to validate", lesson.Slug)
			continue
		}

		log.Infof("Successfully imported lesson '%s'  with %d endpoints.", lesson.Slug, len(lesson.Endpoints))

		retLds = append(retLds, &lesson)
	}

	log.Info("Performing post-import lesson prereq validation check")
	for _, lesson := range retLds {
		for _, prereqSlug := range lesson.Prereqs {
			if !lessonInSlice(retLds, prereqSlug) {
				err := fmt.Errorf("Lesson %s listed an unknown prereq: %s", lesson.Slug, prereqSlug)
				log.Error(err.Error())
				return nil, err
			}
		}
	}

	if len(fileList) != len(retLds) {
		log.Warnf("Imported %d lesson definitions with errors.", len(retLds))
		return retLds, errors.New("Not all lesson definitions were imported")
	}

	log.Infof("Imported %d lesson definitions.", len(retLds))
	return retLds, nil

}

func lessonInSlice(lessons []*models.Lesson, slug string) bool {
	for _, lesson := range lessons {
		if lesson.Slug == slug {
			return true
		}
	}
	return false
}

// validateLesson validates a single lesson, returning a simple error if the lesson fails
// to validate. Note that this function also makes some changes to the provided lesson definition
// in memory, such as additing additional context for internal use
func validateLesson(lesson *models.Lesson) error {

	file := lesson.LessonFile

	// Most of the validation heavy lifting should be done via JSON schema as much as possible.
	// This should be run first, and then only checks that can't be done with JSONschema will follow.
	if !lesson.JSValidate() {
		log.Errorf("Basic schema validation failed on %s - see log for errors.", file)
		return errBasicValidation
	}

	seenEndpoints := map[string]string{}
	for n := range lesson.Endpoints {
		ep := lesson.Endpoints[n]
		if _, ok := seenEndpoints[ep.Name]; ok {
			log.Errorf("Failed to import %s: - Endpoint %s appears more than once", file, ep.Name)
			return errDuplicateEndpoint
		}
		seenEndpoints[ep.Name] = ep.Name
	}

	// Endpoint-specific checks
	for i := range lesson.Endpoints {
		ep := lesson.Endpoints[i]

		// Must EITHER provide additionalPorts, or Presentations. Endpoints without either are invalid.
		if len(ep.Presentations) == 0 && len(ep.AdditionalPorts) == 0 {
			log.Error("No presentations configured, and no additionalPorts specified")
			return errInsufficientPresentation
		}

		// Perform configuration-related checks, if relevant
		if ep.ConfigurationType != "" {

			// Regular expressions for matching recognized config files by type
			fileMap := map[string]string{
				"python":  fmt.Sprintf(`.*%s\.py`, ep.Name),
				"ansible": fmt.Sprintf(`.*%s\.yml`, ep.Name),
				"napalm":  fmt.Sprintf(`.*%s-(junos|eos|iosxr|nxos|nxos_ssh|ios)\.txt$`, ep.Name),
			}

			for s := range lesson.Stages {

				configDir := fmt.Sprintf("%s/stage%d/configs/", filepath.Dir(file), s)
				configFile := ""
				err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
					var validID = regexp.MustCompile(fileMap[ep.ConfigurationType])
					if validID.MatchString(path) {
						configFile = filepath.Base(path)
						return nil
					}
					return nil
				})
				if err != nil {
					log.Error(err)
					return err
				}

				if configFile == "" || configFile == "." {
					log.Errorf("Configuration file for endpoint '%s' was not found.", ep.Name)
					return errMissingConfigurationFile
				}

				lesson.Endpoints[i].ConfigurationFile = configFile
			}
		}

		// Ensure each presentation name is unique for each endpoint
		seenPresentations := map[string]string{}
		for n := range ep.Presentations {
			if _, ok := seenPresentations[ep.Presentations[n].Name]; ok {
				log.Errorf("Failed to import %s: - Presentation %s appears more than once for an endpoint", file, ep.Presentations[n].Name)
				return errDuplicatePresentation
			}
			seenPresentations[ep.Presentations[n].Name] = ep.Presentations[n].Name
		}
	}

	// Ensure all connections are referring to endpoints that are actually present in the definition
	for c := range lesson.Connections {
		connection := lesson.Connections[c]

		if !entityInLabDef(connection.A, lesson) {
			log.Errorf("Failed to import %s: - Connection %s refers to nonexistent entity", file, connection.A)
			return errBadConnection
		}

		if !entityInLabDef(connection.B, lesson) {
			log.Errorf("Failed to import %s: - Connection %s refers to nonexistent entity", file, connection.B)
			return errBadConnection
		}
	}

	// Iterate over stages, and retrieve lesson guide content
	for l := range lesson.Stages {
		s := lesson.Stages[l]

		guideFileMap := map[string]string{
			"markdown": ".md",
			"jupyter":  ".ipynb",
		}

		fileName := fmt.Sprintf("%s/stage%d/guide%s", filepath.Dir(file), l, guideFileMap[string(s.GuideType)])
		contents, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Errorf("Encountered problem reading lesson guide: %s", err)
			return errMissingLessonGuide
		}
		lesson.Stages[l].GuideContents = string(contents)
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
