package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// Image is a resource type that provides metadata for endpoint images in use within Lessons
type Image struct {
	Slug string `json:"Slug" yaml:"slug" jsonschema:"Unique identifier for this image"`

	Description string `json:"Description" yaml:"description" jsonschema:"Description of this image"`

	// Temporary measure to grant privileges to endpoints selectively
	Privileged bool `json:"Privileged" yaml:"privileged" jsonschema:"Should this image be granted admin privileges?"`

	// Used to allow authors to know which interfaces are available, and in which order they'll be connected
	NetworkInterfaces []string `json:"NetworkInterfaces" yaml:"networkInterfaces" jsonschema:"required,minItems=1"`

	SSHUser     string `json:"SSHUser" yaml:"sshUser" jsonschema:"Username for SSH connections"`
	SSHPassword string `json:"SSHPassword" yaml:"sshPassword" jsonschema:"Password for SSH Connections"`
}

// GetSchema returns a Schema to be used in creation wizards
func (i Image) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(i)
}

// JSValidate uses an Antidote resource's struct properties and tags to construct a jsonschema
// document, and then validates that instance's values against that schema.
func (i Image) JSValidate() bool {

	// Load JSON Schema document for type
	schemaReflect := jsonschema.Reflect(i)
	b, _ := json.Marshal(schemaReflect)
	schemaLoader := gjs.NewStringLoader(string(b))
	schema, _ := gjs.NewSchema(schemaLoader)

	// Load instance JSON document
	b, err := json.Marshal(i)
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
