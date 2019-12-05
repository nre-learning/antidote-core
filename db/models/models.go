package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// JSValidate performs a validity check against the JSON encoding of a Lesson struct
// with the exported json schema from that type.
func JSValidate(resource interface{}) bool {

	// Load JSON Schema document for Lesson type
	schemaLoader := gjs.NewStringLoader(GetJSONSchema(resource))
	schema, _ := gjs.NewSchema(schemaLoader)

	// Load lesson instance JSON document
	b, err := json.Marshal(resource)
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

	return result.Valid()
}

// GetJSONSchema is a helper function for generating a JSON Schema document for an Antidote resource.
// Used by internal validation purposes, but also useful to export via command-line for 3rd-party consumption
func GetJSONSchema(resource interface{}) string {
	schema := jsonschema.Reflect(resource)
	b, _ := json.Marshal(schema)
	return string(b)
}
