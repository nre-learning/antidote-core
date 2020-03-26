package db

import (
	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	ot "github.com/opentracing/opentracing-go"
)

// ImportCurriculum provides a single function for all curriculum resources to be imported and placed
// within the backing data store
func ImportCurriculum(dm db.DataManager, config config.AntidoteConfig) error {
	span := ot.StartSpan("ingestor_curriculum_import")
	defer span.Finish()

	// TODO(mierdin): Add step to read in curriculum meta and make sure the calling code
	// (BOTH antidote validate and antidoted import functions)
	// checks that the values are correct, including the targeted antidote version

	collections, err := ReadCollections(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertCollections(span.Context(), collections)

	lessons, err := ReadLessons(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertLessons(span.Context(), lessons)

	images, err := ReadImages(config.CurriculumDir)
	if err != nil {
		// log.Warn(err)
	}
	dm.InsertImages(span.Context(), images)

	return nil
}
