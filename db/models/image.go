package db

import (
	jsonschema "github.com/alecthomas/jsonschema"
)

type Image struct {
	Slug string `json:"Slug" yaml:"slug" jsonschema:"Unique identifier for this image"`

	Description string `json:"Description" yaml:"description" jsonschema:"Description of this image"`

	// Temporary measure to grant privileges to endpoints selectively
	VirtualMachine string `json:"VirtualMachine" yaml:"virtualMachine" jsonschema:"Does this Image house a virtual machine?"`

	// Used to allow authors to know which interfaces are available, and in which order they'll be connected
	NetworkInterfaces []string `json:"NetworkInterfaces" yaml:"networkInterfaces" jsonschema:"required,minItems=1"`
}

// GetSchema returns a Schema to be used in creation wizards
func (l Image) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(l)
}
