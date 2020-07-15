package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// Image is a resource type that provides metadata for endpoint images in use within Lessons
type Image struct {
	Slug string `json:"Slug" yaml:"slug" jsonschema:"description=Unique identifier for this image,pattern=^[A-Za-z0-9\\-]*$"`

	Description string `json:"Description" yaml:"description" jsonschema:"description=Description of this image"`

	// - "trusted" - regular container on the default runtime (i.e. runc), running in privileged mode. Should **only** be used sparingly, and only for images with its own virtualization layer
	// - "untrusted" - provisioned with the kata runtimeclass, with no privileges or additional capabilities
	Flavor ImageFlavor `json:"Flavor" yaml:"flavor" jsonschema:"required,enum=trusted,enum=untrusted"`

	// This only enables forwarding at the container level. If this image uses the trusted flavor and is running a totally separate qemu VM, this will not affect the inner OS.
	// Kata will forward sysctl calls, so this is mainly targeted at untrusted images that need to forward https://github.com/kata-containers/runtime/issues/185
	EnableForwarding bool `json:"EnableForwarding" yaml:"enableForwarding" jsonschema:"description=Enable IP (v4 and v6) forwarding for this image at runtime"`

	// Used to allow authors to know which interfaces are available, and in which order they'll be connected
	NetworkInterfaces []string `json:"NetworkInterfaces" yaml:"networkInterfaces" jsonschema:"minItems=1"`

	SSHUser     string `json:"SSHUser" yaml:"sshUser" jsonschema:"minLength=1,description=Username for SSH connections"`
	SSHPassword string `json:"SSHPassword" yaml:"sshPassword" jsonschema:"minLength=1,Password for SSH Connections"`

	ConfigUser     string `json:"ConfigUser" yaml:"configUser" jsonschema:"minLength=1,description=Username for configuration scripts"`
	ConfigPassword string `json:"ConfigPassword" yaml:"configPassword" jsonschema:"minLength=1,Password for configuration scripts"`
}

type ImageFlavor string

const (
	FlavorTrusted   ImageFlavor = "trusted"
	FlavorUntrusted ImageFlavor = "untrusted"
)

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
