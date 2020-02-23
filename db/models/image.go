package db

import (
	jsonschema "github.com/alecthomas/jsonschema"
)

// Image is a resource type that provides metadata for endpoint images in use within Lessons
type Image struct {
	Slug string `json:"Slug" yaml:"slug" jsonschema:"Unique identifier for this image"`

	Description string `json:"Description" yaml:"description" jsonschema:"Description of this image"`

	// Temporary measure to grant privileges to endpoints selectively
	Privileged bool `json:"Privileged" yaml:"privileged" jsonschema:"Should this image be granted admin privileges?"`

	// Used to allow authors to know which interfaces are available, and in which order they'll be connected
	NetworkInterfaces []string `json:"NetworkInterfaces" yaml:"networkInterfaces" jsonschema:"required,minItems=1"`
}

// GetSchema returns a Schema to be used in creation wizards
func (l Image) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(l)
}
