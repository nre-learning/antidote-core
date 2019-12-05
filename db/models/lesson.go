package db

import (
	"encoding/json"
)

func (l *Lesson) JSON() string {
	b, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	return string(b)
}

type Lesson struct {

	Id int32 `json:"Id sql:",pk"`

	// While Id will be used as a true unique identifier, the Slug is now what we'll use
	// to look up this lesson.
	Slug          string              `json:"Slug" yaml:"slug" pg:",unique"`

	Stages        []*LessonStage      `json:"Stages,omitempty" yaml:"stages,omitempty" jsonschema:"required"`
	LessonName    string              `json:"LessonName,omitempty" yaml:"lessonName,omitempty" jsonschema:"required"`
	Endpoints     []*LessonEndpoint   `json:"Endpoints,omitempty" yaml:"endpoints,omitempty" jsonschema:"required"`
	Connections   []*LessonConnection `json:"Connections,omitempty" yaml:"connections,omitempty"`
	Category      string              `json:"Category,omitempty" yaml:"category,omitempty" jsonschema:"required"`
	LessonDiagram string              `json:"LessonDiagram,omitempty" yaml:"lessonDiagram,omitempty"`
	LessonVideo   string              `json:"LessonVideo,omitempty" yaml:"lessonVideo,omitempty"`
	Tier          string              `json:"Tier,omitempty" yaml:"tier,omitempty" jsonschema:"required"`
	Prereqs       []int32             `json:"Prereqs,omitempty" yaml:"prereqs,omitempty"`
	Tags          []string            `json:"Tags,omitempty" yaml:"tags,omitempty"`
	Collection    int32               `json:"Collection,omitempty" yaml:"collection,omitempty"`
	Description   string              `json:"Description,omitempty" yaml:"description,omitempty" jsonschema:"required"`

	// TODO(mierdin): Figure out if these are needed anymore.
	LessonFile string `json:"LessonFile,omitempty" yaml:"lessonFile,omitempty"`
	LessonDir  string `json:"LessonDir,omitempty" yaml:"lessonDir,omitempty"`
}

type LessonStage struct {

	Id int32 `json:"Id sql:",pk"`

	Description        string `json:"Description,omitempty" yaml:"description,omitempty"`
	LabGuide           string `json:"LabGuide,omitempty" yaml:"labGuide,omitempty"`
	JupyterLabGuide    bool   `json:"JupyterLabGuide,omitempty" yaml:"jupyterLabGuide,omitempty"`
	VerifyCompleteness bool   `json:"VerifyCompleteness,omitempty" yaml:"verifyCompleteness,omitempty"`
	VerifyObjective    string `json:"VerifyObjective,omitempty" yaml:"verifyObjective,omitempty"`
}

type LessonEndpoint struct {
	Name  string `json:"Name,omitempty" yaml:"name,omitempty"`
	Image string `json:"Image,omitempty" yaml:"image,omitempty"`

	// Validation for this field will be done post-validation
	ConfigurationType string `json:"ConfigurationType,omitempty" yaml:"configurationType,omitempty"`

	// Handles any ports not explicitly mentioned in a presentation
	AdditionalPorts []int32 `json:"AdditionalPorts,omitempty" yaml:"additionalPorts,omitempty"`

	Presentations []*LessonPresentation `json:"Presentations,omitempty" yaml:"presentations,omitempty"`
	Host          string                `json:"Host,omitempty" yaml:"host,omitempty"`
}

type LessonPresentation struct {
	Name string `json:"Name,omitempty" yaml:"name,omitempty"`
	Port int32  `json:"Port,omitempty" yaml:"port,omitempty"`
	Type string `json:"Type,omitempty" yaml:"type,omitempty"`
}

type LessonConnection struct {
	A string `json:"A,omitempty" yaml:"a,omitempty"`
	B string `json:"B,omitempty" yaml:"b,omitempty"`
}
