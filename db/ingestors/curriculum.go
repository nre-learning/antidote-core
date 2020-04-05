package db

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	ot "github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ImportCurriculum provides a single function for all curriculum resources to be imported and placed
// within the backing data store
func ImportCurriculum(dm db.DataManager, cfg config.AntidoteConfig) error {
	span := ot.StartSpan("ingestor_curriculum_import")
	defer span.Finish()

	// TODO(mierdin): Enforce a version check with the curriculum when loaded
	curriculum, err := ReadCurriculum(cfg)
	if err != nil {
		return err
	}
	dm.SetCurriculum(span.Context(), curriculum)

	collections, err := ReadCollections(cfg)
	if err != nil {
		return err
	}
	dm.InsertCollections(span.Context(), collections)

	images, err := ReadImages(cfg)
	if err != nil {
		return err
	}
	dm.InsertImages(span.Context(), images)

	lessons, err := ReadLessons(cfg)
	if err != nil {
		return err
	}
	dm.InsertLessons(span.Context(), lessons)

	return nil
}

func ReadCurriculum(cfg config.AntidoteConfig) (*models.Curriculum, error) {

	curriculumFilePath := fmt.Sprintf("%s/curriculum.meta.yaml", cfg.CurriculumDir)
	log.Infof("Attempting to read curriculum information from %s", curriculumFilePath)

	if _, err := os.Stat(curriculumFilePath); err != nil {
		log.Error(err)
		return nil, err
	}

	yamlDef, err := ioutil.ReadFile(curriculumFilePath)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var curriculum models.Curriculum
	err = yaml.Unmarshal([]byte(yamlDef), &curriculum)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !curriculum.JSValidate() {
		log.Errorf("Basic schema validation failed on %s - see log for errors.", curriculum.Name)
		return nil, errBasicValidation
	}

	log.Info("Curriculum information loaded.")

	return &curriculum, nil
}
