package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// Curriculum is a resource type that defines a bit of meta-data for a Curriculum as a whole.
type Curriculum struct {
	Name        string `json:"Name" yaml:"name" jsonschema:"minLength=1,description=Name of this curriculum"`
	Description string `json:"Description" yaml:"description" jsonschema:"minLength=1,description=Description of this curriculum"`
	Website     string `json:"Website" yaml:"website" jsonschema:"minLength=1,description=Website for this curriculum"`
	AVer        string `json:"AVer" yaml:"aVer" jsonschema:"minLength=1,description=The version of Antidote this curriculum was built for"`
	GitRoot     string `json:"GitRoot" yaml:"gitRoot" jsonschema:"minLength=1,description=The web URL to the repository that houses this curriculum"`
}

// JSValidate uses an Antidote resource's struct properties and tags to construct a jsonschema
// document, and then validates that instance's values against that schema.
func (c Curriculum) JSValidate() bool {

	// Load JSON Schema document for type
	schemaReflect := jsonschema.Reflect(c)
	b, _ := json.Marshal(schemaReflect)
	schemaLoader := gjs.NewStringLoader(string(b))
	schema, _ := gjs.NewSchema(schemaLoader)

	// Load instance JSON document
	b, err := json.Marshal(c)
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
	for j := range validationErrors {
		log.Errorf("Validation error - %s", validationErrors[j].String())
	}

	return result.Valid()
}
