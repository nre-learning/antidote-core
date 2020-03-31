package db

import (
	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	ot "github.com/opentracing/opentracing-go"
)

// ImportCurriculum provides a single function for all curriculum resources to be imported and placed
// within the backing data store
func ImportCurriculum(dm db.DataManager, cfg config.AntidoteConfig) error {
	span := ot.StartSpan("ingestor_curriculum_import")
	defer span.Finish()

	// There is a model for a Curriculum type, but we're still figuring out if/how we want to
	// use that, so I'm leaving it out for now. This is where we would likely import it, and perhaps also
	// do checks like version compatibility with Antidote version, etc.

	collections, err := ReadCollections(cfg)
	if err != nil {
		return err
	}
	dm.InsertCollections(span.Context(), collections)

	lessons, err := ReadLessons(cfg)
	if err != nil {
		return err
	}
	dm.InsertLessons(span.Context(), lessons)

	images, err := ReadImages(cfg)
	if err != nil {
		return err
	}
	dm.InsertImages(span.Context(), images)

	return nil
}
