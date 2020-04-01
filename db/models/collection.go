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
	Slug             string `json:"Slug" yaml:"slug" jsonschema:"minLength=1,description=A unique identifier for the lesson, usually 2-3 words with hyphens,pattern=^[a-z-]*$"`
	Title            string `json:"Title" yaml:"title" jsonschema:"minLength=1,description=A human-readable name for the collection (i.e. a company name)"`
	Image            string `json:"Image" yaml:"image" jsonschema:"minLength=1,description=An internet-accessible URL to a logo for this collection"`
	Website          string `json:"Website" yaml:"website" jsonschema:"minLength=1,description=URL to the website for the person or organization this collection represents"`
	ContactEmail     string `json:"ContactEmail" yaml:"contactEmail" jsonschema:"description=Contact email address for this collection"`
	BriefDescription string `json:"BriefDescription" yaml:"briefDescription" jsonschema:"minLength=1,description=A brief description of the collection"`
	LongDescription  string `json:"LongDescription" yaml:"longDescription" jsonschema:"minLength=1,description=A longer-form description of the collection, which may include explanations of the person or organization sponsoring it"`
	Type             string `json:"Type" yaml:"type" jsonschema:"minLength=1,description=The type of collection,enum=vendor,enum=community,enum=consultancy"`
	Tier             string `json:"Tier" yaml:"tier" jsonschema:"description=Tier for this collection (you probably want 'prod') ,enum=prod,enum=ptr,enum=local"`

	CollectionFile string `json:"-" jsonschema:"-"`
}

// GetSchema returns a Schema to be used in creation wizards
func (c Collection) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(c)
}

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
