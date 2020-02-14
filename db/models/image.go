package db

import (
	jsonschema "github.com/alecthomas/jsonschema"
)

type Image struct {
	Name string `json:"Slug" yaml:"slug" jsonschema:"Unique identifier for this image"`

	// Temporary measure to grant privileges to endpoints selectively
	VirtualMachine string `json:"Slug" yaml:"slug" jsonschema:"Does this Image house a virtual machine?"`

	// Stages []*LessonStage `json:"Stages" yaml:"stages" jsonschema:"required,minItems=1"`

	// Used to allow authors to know which interfaces are available, and in which order they'll be connected
	NetworkInterfaces []string `json:"NetworkInterfaces" yaml:"stages" jsonschema:"required,minItems=1"`
}

// GetSchema returns a Schema to be used in creation wizards
func (l Image) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(l)
}
