package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ReadCollections reads collection definitions from the filesystem, validates them, and returns them
// in a slice.
func ReadCollections(cfg config.AntidoteConfig) ([]*models.Collection, error) {

	fileList := []string{}
	collectionDir := fmt.Sprintf("%s/collections", cfg.CurriculumDir)
	log.Debugf("Searching %s for collection definitions", collectionDir)
	err := filepath.Walk(collectionDir, func(path string, f os.FileInfo, err error) error {
		colFile := fmt.Sprintf("%s/collection.meta.yaml", path)
		if _, err := os.Stat(colFile); err == nil {
			log.Debugf("Found collection definition at: %s", colFile)
			fileList = append(fileList, colFile)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	retCollections := []*models.Collection{}

	for f := range fileList {

		file := fileList[f]

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue
		}

		var collection models.Collection
		err = yaml.Unmarshal([]byte(yamlDef), &collection)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
		}
		collection.CollectionFile = file

		err = validateCollection(&collection)
		if err != nil {
			continue
		}

		if tierMap[collection.Tier] < tierMap[cfg.Tier] {
			log.Warnf("Skipping collection %s due to configured tier", collection.Slug)
			continue
		}

		log.Infof("Successfully imported collection %s: %s", collection.Slug, collection.Title)
		retCollections = append(retCollections, &collection)
	}

	if len(fileList) == len(retCollections) {
		log.Infof("Imported %d collection definitions.", len(retCollections))
		return retCollections, nil
	} else {
		log.Warnf("Imported %d collection definitions with errors.", len(retCollections))
		return retCollections, errors.New("Not all collection definitions were imported")
	}

	return retCollections, nil

}

// validateCollection validates a single collection, returning a simple error if the collection fails
// to validate.
func validateCollection(collection *models.Collection) error {

	file := collection.CollectionFile

	// Basic validation from jsonschema
	if !collection.JSValidate() {
		log.Errorf("Basic schema validation failed on %s - see log for errors.", file)
		return errBasicValidation
	}

	return nil
}
