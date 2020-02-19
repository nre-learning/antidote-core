package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// Lesson represents the fields and sub-types for defining a lesson resource in Antidote
// Only this struct should be loaded as a table. All sub-values can be stored as binary JSON
// and deserialized quickly upon retrieval.
type Lesson struct {
	Slug string `json:"Slug" yaml:"slug" sql:",pk" pg:",unique" jsonschema:"description=Unique slug to identify this lesson"`

	Stages      []*LessonStage      `json:"Stages" yaml:"stages" jsonschema:"required,minItems=1"`
	Name        string              `json:"Name" yaml:"name" jsonschema:"required,description=Name of the lesson"`
	Endpoints   []*LessonEndpoint   `json:"Endpoints" yaml:"endpoints" jsonschema:"required,minItems=1"`
	Connections []*LessonConnection `json:"Connections" yaml:"connections"`
	Category    string              `json:"Category" yaml:"category" jsonschema:"required,description=Category for the lesson"`
	Diagram     string              `json:"Diagram" yaml:"diagram" jsonschema:"description=URL to lesson diagram"`
	Video       string              `json:"Video" yaml:"video" jsonschema:"description=URL to lesson video"`
	Tier        string              `json:"Tier" yaml:"tier" jsonschema:"required,description=Tier for this lesson,pattern=local|ptr|prod"`
	Prereqs     []string            `json:"Prereqs,omitempty" yaml:"prereqs"`
	Tags        []string            `json:"Tags" yaml:"tags"`
	Collection  int32               `json:"Collection" yaml:"collection"`
	Description string              `json:"Description" yaml:"description" jsonschema:"required,description=Description of this lesson"`

	// TODO(mierdin): Figure out if these are needed anymore.
	LessonFile string `json:"-" jsonschema:"-"`
	LessonDir  string `json:"-" jsonschema:"-"`
}

// GetSchema returns a Schema to be used in creation wizards
func (l Lesson) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(l)
}

// TODO(mierdin): Implement this
func sortSchema(js *jsonschema.Schema) *jsonschema.Schema {
	return js
}

// JSValidate uses an Antidote resource's struct properties and tags to construct a jsonschema
// document, and then validates that instance's values against that schema.
func (l Lesson) JSValidate() bool {

	// Load JSON Schema document for Lesson type
	schemaReflect := jsonschema.Reflect(l)
	b, _ := json.Marshal(schemaReflect)
	schemaLoader := gjs.NewStringLoader(string(b))
	schema, _ := gjs.NewSchema(schemaLoader)

	// Load lesson instance JSON document
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

// LessonStage is a specific state that a Lesson can be in. This can be thought of like chapters in a book.
// A Lesson might have one or more LessonStages.
type LessonStage struct {
	Description   string `json:"Description" yaml:"description"`
	GuideType     string `json:"GuideType" yaml:"guideType" jsonschema:"required,pattern=jupyter|markdown"`
	GuideContents string `json:"GuideContents,omitempty" jsonschema:"-"`

	// TODO(mierdin): Implementing this later
	// Objectives    []*LessonStageObjective `json:"Objectives,omitempty" yaml:"objectives"`
}

// TODO(mierdin): Implementing this later
// type LessonStageObjective struct {
// 	ID            int32               `json:"ID,omitempty"`
// 	Description string `json:"Description" yaml:"description" jsonschema:"required"`
// }

// LessonEndpoint is typically a container that runs some software in a Lesson. This can be a network device,
// or a simple container with some Python libraries installed - it doesn't really matter. It's just some software
// that you want to have running in a lesson for educational purposes
type LessonEndpoint struct {
	Name  string `json:"Name" yaml:"name" jsonschema:"description=Name of the endpoint"`
	Image string `json:"Image" yaml:"image" jsonschema:"description=Container image reference for the endpoint,pattern=^[A-Za-z0-9/-]*$"`

	ConfigurationType string `json:"ConfigurationType,omitempty" yaml:"configurationType" jsonschema:"pattern=napalm-.*|python|ansible"`

	AdditionalPorts []int32 `json:"AdditionalPorts,omitempty" yaml:"additionalPorts" jsonschema:"description=Additional ports to open that aren't in a Presentation"`

	Presentations []*LessonPresentation `json:"Presentations,omitempty" yaml:"presentations"`
}

// LessonPresentation is a particular view into a LessonEndpoint. It's a way of specifying how an endpoint
// should be made available to the user in the front-end
type LessonPresentation struct {
	Name string `json:"Name" yaml:"name" jsonschema:"required"`
	Port int32  `json:"Port" yaml:"port" jsonschema:"required,minimum=1"`
	Type string `json:"Type" yaml:"type" jsonschema:"required,pattern=http|ssh"`
}

// LessonConnection is a point-to-point network connection between two LessonEndpoints. The `A` and `B` properties should
// refer to the Name of LessonEndpoints within a given lesson that should be networked together.
type LessonConnection struct {
	A string `json:"A" yaml:"a" jsonschema:"required"`
	B string `json:"B" yaml:"b" jsonschema:"required"`
}
