package db

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
)

// ReadImages reads image definitions from the filesystem, validates them, and returns them
// in a slice.
func ReadImages(cfg config.AntidoteConfig) ([]*models.Image, error) {

	// Get image definitions
	fileList := []string{}
	imageDir := fmt.Sprintf("%s/images", cfg.CurriculumDir)
	log.Debugf("Searching %s for image definitions", imageDir)
	err := filepath.Walk(imageDir, func(path string, f os.FileInfo, err error) error {
		imageLoc := fmt.Sprintf("%s/image.meta.yaml", path)
		if _, err := os.Stat(imageLoc); err == nil {
			log.Debugf("Found image definition at: %s", imageLoc)
			fileList = append(fileList, imageLoc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	images := []*models.Image{}

	for f := range fileList {

		file := fileList[f]

		log.Infof("Importing image definition at: %s", file)

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue
		}

		var image models.Image
		err = yaml.Unmarshal([]byte(yamlDef), &image)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
		}

		err = validateImage(&image)
		if err != nil {
			log.Errorf("Image '%s' failed to validate", image.Slug)
			continue
		}

		log.Infof("Successfully imported image '%s'", image.Slug)

		images = append(images, &image)
	}

	if len(fileList) == len(images) {
		log.Infof("Imported %d image definitions.", len(images))
		return images, nil
	}

	log.Warnf("Imported %d image definitions with errors.", len(images))
	return images, errors.New("Not all image definitions were imported")
}

func validateImage(image *models.Image) error {

	// Most of the validation heavy lifting should be done via JSON schema as much as possible.
	// This should be run first, and then only checks that can't be done with JSONschema will follow.
	if !image.JSValidate() {
		log.Errorf("Basic schema validation failed on %s - see log for errors.", image.Slug)
		return errBasicValidation
	}

	for i := range image.NetworkInterfaces {
		if image.NetworkInterfaces[i] == "eth0" {
			log.Error("No presentations configured, and no additionalPorts specified")
			return errEth0NotAllowed
		}
	}

	return nil
}
