package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

type Lesson struct {
	Id int32 `json:"Id" sql:",pk"`

	// While Id will be used as a true unique identifier, the Slug is now what we'll use
	// to look up this lesson.
	Slug string `json:"Slug" yaml:"slug" pg:",unique"`

	Stages        []*LessonStage      `json:"Stages" yaml:"stages" jsonschema:"required,minItems=1"`
	LessonName    string              `json:"LessonName" yaml:"lessonName" jsonschema:"required"`
	Endpoints     []*LessonEndpoint   `json:"Endpoints" yaml:"endpoints" jsonschema:"required,minItems=1"`
	Connections   []*LessonConnection `json:"Connections" yaml:"connections"`
	Category      string              `json:"Category" yaml:"category" jsonschema:"required"`
	LessonDiagram string              `json:"LessonDiagram" yaml:"lessonDiagram"`
	LessonVideo   string              `json:"LessonVideo" yaml:"lessonVideo"`
	Tier          string              `json:"Tier" yaml:"tier" jsonschema:"required" jsonschema:"required,pattern=local|ptr|prod"`
	Prereqs       []string            `json:"Prereqs,omitempty" yaml:"prereqs"`
	Tags          []string            `json:"Tags" yaml:"tags"`
	Collection    int32               `json:"Collection" yaml:"collection"`
	Description   string              `json:"Description" yaml:"description" jsonschema:"required"`

	// TODO(mierdin): Figure out if these are needed anymore.
	LessonFile string `json:"-" jsonschema:"-"`
	LessonDir  string `json:"-" jsonschema:"-"`
}

func (l *Lesson) JSON() string {
	b, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	return string(b)
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

type LessonStage struct {
	Id int32 `json:"Id" sql:",pk"`

	Description string                  `json:"Description" yaml:"description"`
	GuideType   string                  `json:"GuideType" yaml:"guideType" jsonschema:"required,pattern=jupyter|markdown"`
	LabGuide    string                  `json:"LabGuide,omitempty" jsonschema:"-"`
	Objectives  []*LessonStageObjective `json:"Objectives,omitempty" yaml:"objectives"`
}

type LessonStageObjective struct {
	Id          int32  `json:"Id" sql:",pk"`
	Description string `json:"Description" yaml:"description" jsonschema:"required"`
}

type LessonEndpoint struct {
	Name  string `json:"Name" yaml:"name" jsonschema:"description=Name of the endpoint"`
	Image string `json:"Image" yaml:"image" jsonschema:"description=Container image reference for the endpoint,pattern=^[A-Za-z0-9/\-]*$"`

	ConfigurationType string `json:"ConfigurationType,omitempty" yaml:"configurationType" jsonschema:"pattern=napalm-.*|python|ansible"`

	AdditionalPorts []int32 `json:"AdditionalPorts,omitempty" yaml:"additionalPorts" jsonschema:"description=Additional ports to open that aren't in a Presentation"`

	Presentations []*LessonPresentation `json:"Presentations,omitempty" yaml:"presentations"`
}

type LessonPresentation struct {
	Name string `json:"Name" yaml:"name" jsonschema:"required"`
	Port int32  `json:"Port" yaml:"port" jsonschema:"required,minimum=1"`
	Type string `json:"Type" yaml:"type" jsonschema:"required,pattern=http|ssh"`
}

type LessonConnection struct {
	A string `json:"A" yaml:"a" jsonschema:"required"`
	B string `json:"B" yaml:"b" jsonschema:"required"`
}
