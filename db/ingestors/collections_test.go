package db

import (
	"testing"

	"github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
)

// getValidCollection returns a full, valid example of an Collection that uses all the features.
// Tests in this file should make use of this by making a copy, tweaking in some way that makes it
// invalid, and then asserting on the error type/message.
func getValidCollection() models.Collection {
	collections, err := ReadCollections(config.AntidoteConfig{
		CurriculumDir: "../test/test-curriculum",
		Tier:          "local",
	})
	if err != nil {
		panic(err)
	}
	for _, c := range collections {
		if c.Slug == "valid-collection" {
			return *c
		}
	}
	panic("unable to find valid collection")
}

func TestValidCollection(t *testing.T) {
	c := getValidCollection()
	err := validateCollection(&c)
	assert(t, (err == nil), "Expected validation to pass, but encountered validation errors")
}

func TestBadCollectionSlug(t *testing.T) {
	c := getValidCollection()
	c.Slug = "foobar9b3#$(*#"
	err := validateCollection(&c)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}
