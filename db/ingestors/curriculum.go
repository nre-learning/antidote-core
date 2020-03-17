package db

import (
	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
)

// ImportCurriculum provides a single function for all curriculum resources to be imported and placed
// within the backing data store
func ImportCurriculum(dm db.DataManager, config config.AntidoteConfig) error {

	// TODO(mierdin): Add step to read in curriculum meta and make sure the calling code
	// (BOTH antidote validate and antidoted import functions)
	// checks that the values are correct, including the targeted antidote version

	collections, err := ReadCollections(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertCollections(collections)

	lessons, err := ReadLessons(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertLessons(lessons)

	images, err := ReadImages(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertImages(images)

	return nil
}
