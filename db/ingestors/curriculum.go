package db

import (
	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
)

// ImportCurriculum provides a single function for all curriculum resources to be imported and placed
// within the backing data store
func ImportCurriculum(dm db.DataManager, config config.AntidoteConfig) error {

	// collections, err := ReadCollections(config.CurriculumDir)
	// if err != nil {
	// 	// log.Warn(err)
	// }
	// dm.InsertCollections(collections)

	lessons, err := ReadLessons(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertLessons(lessons)

	return nil
}
