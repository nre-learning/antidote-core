package db

import (
	"errors"
	"fmt"
	"strings"

	pg "github.com/go-pg/pg"
	orm "github.com/go-pg/pg/orm"
	log "github.com/sirupsen/logrus"

	config "github.com/nre-learning/syringe/config"
	models "github.com/nre-learning/syringe/db/models"
)

// DataManager enforces the set of functions required by the rest of the Antidote codebase for
// handling internal data, such as live state or loaded curriculum resource definitions.
type DataManager interface {

	// Misc
	Preflight() error
	Initialize() error

	// Images
	// TODO(mierdin): You may want to consider having ReadLessons in a separate
	// place - as it doesn't interact with the database at all.
	// ReadLessons() ([]*models.Lesson, error)
	// InsertLessons([]*models.Lesson) error
	// ListLessons() ([]*models.Lesson, error)
	// GetLesson(string) (*models.Lesson, error)

	// Lessons
	// TODO(mierdin): You may want to consider having ReadLessons in a separate
	// place - as it doesn't interact with the database at all.
	ReadLessons() ([]*models.Lesson, error)
	InsertLessons([]*models.Lesson) error
	ListLessons() ([]*models.Lesson, error)
	GetLesson(string) (*models.Lesson, error)

	// // Collections
	// TODO(mierdin): You may want to consider having ReadCollections in a separate
	// place - as it doesn't interact with the database at all.
	// ReadCollections() error
	// InsertCollection([]*models.Collection) error
	// ListCollections() ([]*models.Collection, error)
	// GetCollection(string) (*models.Collection, error)
	// DeleteCollection(string) error

	// // Curriculum
	// SetCurriculum(*models.Curriculum) error

	// // LiveLessons
	// CreateLiveLesson(*models.LiveLesson) error
	// ListLiveLessons() ([]*models.LiveLesson, error)
	// GetLiveLesson(string) (*models.LiveLesson, error)
	// UpdateLiveLesson(*models.LiveLesson) error
	// DeleteLiveLesson(string) error

	// GCWhiteList

	// Sessions
}

// AntidoteData is a specific implementation of DataManager - meant to provide functions for handling
// data within Antidote. This uses postgres as a back-end datastore where appropriate.
type AntidoteData struct {
	User            string
	Password        string
	Database        string
	AntidoteVersion string
	SyringeConfig   *config.SyringeConfig
}

var _ DataManager = &AntidoteData{}

// Preflight is a basic database health checker function for Antidote. It checks
// that the database exists, tables are in place, and that the version that initialized
// this database matches the one that we're operating with now.
func (a *AntidoteData) Preflight() error {
	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	var metaRaw []models.Meta
	_, err := db.Query(&metaRaw, `SELECT * FROM meta`)
	if err != nil {
		return err
	}

	meta := map[string]string{}
	for i := range metaRaw {
		meta[metaRaw[i].Key] = metaRaw[i].Value
	}

	if _, ok := meta["AntidoteVersion"]; ok {
		if meta["AntidoteVersion"] != a.AntidoteVersion {
			return fmt.Errorf("Database provisioned with different version of Antidote (expected %s, got %s). Re-run 'antidote import'", a.AntidoteVersion, meta["AntidoteVersion"])
		}
	} else {
		return errors.New("Unable to retrieve version of database. Re-run 'antidote import'")
	}

	return nil

}

// Initialize drops all Antidote tables, and re-initializes them
func (a *AntidoteData) Initialize() error {

	// Connect to Postgres
	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	// TODO(mierdin): Can we create database first, or at least check if it exists?

	// TODO(mierdin): Acquire database lock here

	for _, model := range []interface{}{
		(*models.Meta)(nil),
		(*models.Lesson)(nil),
		// TODO(mierdin): Don't forget to add the rest of the models, or perhaps find a way to do this dynamically
	} {
		err := db.DropTable(model, &orm.DropTableOptions{
			// Temp: true,
		})
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return err
		}
	}

	for _, model := range []interface{}{
		(*models.Meta)(nil),
		(*models.Lesson)(nil),
	} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			// Temp: true,
		})
		if err != nil {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	metaMap := map[string]string{
		"AntidoteVersion": a.AntidoteVersion,
	}
	for k, v := range metaMap {
		meta := &models.Meta{
			Key:   k,
			Value: v,
		}
		// log.Info("Inserting into meta: %v", meta)
		err := tx.Insert(meta)
		if err != nil {
			log.Errorf("Failed to insert meta information '%s' into the database: %v", meta.Key, err)
			return err
		}
	}
	tx.Commit()

	return nil

}
