package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// Collection is a resource type that provides a type of categorization for other curriculum resources
// like Lessons. A Collection might be defined for a company, an open-source project, or even for an individual,
// as a home for all curriculum resources with strong relationships to that entity, and as a way of giving
// more information for that entity.
type Collection struct {
	Slug             string `json:"Id" yaml:"id"`
	Title            string `json:"Title" yaml:"title"`
	Image            string `json:"Image" yaml:"image"`
	Website          string `json:"Website" yaml:"website"`
	ContactEmail     string `json:"ContactEmail" yaml:"contactEmail"`
	BriefDescription string `json:"BriefDescription" yaml:"briefDescription"`
	LongDescription  string `json:"LongDescription" yaml:"longDescription"`
	Type             string `json:"Type" yaml:"type"`
	Tier             string `json:"Tier" yaml:"tier"`
	CollectionFile   string `json:"CollectionFile" yaml:"collectionFile"`
}

// TODO(mierdin): Add json schema stuff here

// JSValidate uses an Antidote resource's struct properties and tags to construct a jsonschema
// document, and then validates that instance's values against that schema.
func (l Collection) JSValidate() bool {

	// Load JSON Schema document
	schemaReflect := jsonschema.Reflect(l)
	b, _ := json.Marshal(schemaReflect)
	schemaLoader := gjs.NewStringLoader(string(b))
	schema, _ := gjs.NewSchema(schemaLoader)

	// Load instance JSON document
	b, err := json.Marshal(l)
	if err != nil {
		log.Error(err)
		return false
	}
	documentLoader := gjs.NewStringLoader(string(b))

	// Perform validation
	result, err := schema.Validate(documentLoader)
	if err != nil {
		log.Error(err)
		return false
	}

	validationErrors := result.Errors()
	for i := range validationErrors {
		log.Errorf("Validation error - %s", validationErrors[i].String())
	}

	return result.Valid()
}
