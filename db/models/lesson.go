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

	// tableName is an optional field that specifies custom table name and alias.
	// By default go-pg generates table name and alias from struct name.
	// tableName struct{} `pg:"lesson,alias:lessons"` // default values are the same

	// TODO(mierdin): users should not have to create the ID.
	Id int32 `json:"Id,omitempty" yaml:"lessonId,omitempty"`

	Stages        []*LessonStage `json:"Stages,omitempty" yaml:"stages,omitempty" jsonschema:"required"`
	LessonName    string         `json:"LessonName,omitempty" yaml:"lessonName,omitempty" jsonschema:"required"`
	Endpoints     []*Endpoint    `json:"Endpoints,omitempty" yaml:"endpoints,omitempty" jsonschema:"required"`
	Connections   []*Connection  `json:"Connections,omitempty" yaml:"connections,omitempty"`
	Category      string         `json:"Category,omitempty" yaml:"category,omitempty" jsonschema:"required"`
	LessonDiagram string         `json:"LessonDiagram,omitempty" yaml:"lessonDiagram,omitempty"`
	LessonVideo   string         `json:"LessonVideo,omitempty" yaml:"lessonVideo,omitempty"`
	Tier          string         `json:"Tier,omitempty" yaml:"tier,omitempty" jsonschema:"required"`
	Prereqs       []int32        `json:"Prereqs,omitempty" yaml:"prereqs,omitempty"`
	Tags          []string       `json:"Tags,omitempty" yaml:"tags,omitempty"`
	Collection    int32          `json:"Collection,omitempty" yaml:"collection,omitempty"`
	Description   string         `json:"Description,omitempty" yaml:"description,omitempty" jsonschema:"required"`

	// This is meant to fill: "How well do you know <slug>?"
	Slug string `json:"Slug,omitempty" yaml:"slug,omitempty"`

	// TODO(mierdin): Figure out if these are needed anymore.
	LessonFile string `json:"LessonFile,omitempty" yaml:"lessonFile,omitempty"`
	LessonDir  string `json:"LessonDir,omitempty" yaml:"lessonDir,omitempty"`
}

type LessonStage struct {

	// TODO(mierdin): Again this should be auto-generated, lesson authors shouldn't care about this. The order will
	// need to be preserved.
	Id int32 `json:"Id,omitempty" yaml:"id,omitempty"`

	Description        string `json:"Description,omitempty" yaml:"description,omitempty"`
	LabGuide           string `json:"LabGuide,omitempty" yaml:"labGuide,omitempty"`
	JupyterLabGuide    bool   `json:"JupyterLabGuide,omitempty" yaml:"jupyterLabGuide,omitempty"`
	VerifyCompleteness bool   `json:"VerifyCompleteness,omitempty" yaml:"verifyCompleteness,omitempty"`
	VerifyObjective    string `json:"VerifyObjective,omitempty" yaml:"verifyObjective,omitempty"`
}

type Endpoint struct {
	Name  string `json:"Name,omitempty" yaml:"name,omitempty"`
	Image string `json:"Image,omitempty" yaml:"image,omitempty"`

	// Validation for this field will be done post-validation
	ConfigurationType string `json:"ConfigurationType,omitempty" yaml:"configurationType,omitempty"`

	// Handles any ports not explicitly mentioned in a presentation
	AdditionalPorts []int32 `json:"AdditionalPorts,omitempty" yaml:"additionalPorts,omitempty"`

	Presentations []*Presentation `json:"Presentations,omitempty" yaml:"presentations,omitempty"`
	Host          string          `json:"Host,omitempty" yaml:"host,omitempty"`
}

type Presentation struct {
	Name string `json:"Name,omitempty" yaml:"name,omitempty"`
	Port int32  `json:"Port,omitempty" yaml:"port,omitempty"`
	Type string `json:"Type,omitempty" yaml:"type,omitempty"`
}

type Connection struct {
	A string `json:"A,omitempty" yaml:"a,omitempty"`
	B string `json:"B,omitempty" yaml:"b,omitempty"`
}
