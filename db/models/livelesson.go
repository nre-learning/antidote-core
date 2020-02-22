package db

import (
	"github.com/golang/protobuf/ptypes/timestamp"
)

// JSON is a helpoer function to convert the
// func (l *LiveLesson) JSON() string {
// 	b, err := json.Marshal(l)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return string(b)
// }

// LiveLesson is a runtime instance of a Lesson in use. It represents a specific lesson (via LessonID)
// being requested by a specific session (via SessionID) and holds all of the runtime state that Antidote
// needs to know about it to serve it to the front-end appropriately
type LiveLesson struct {
	ID              string                   `json:"LiveLessonId,omitempty"`
	SessionID       string                   `json:"SessionId,omitempty"`
	LessonSlug      string                   `json:"LessonSlug,omitempty"`
	LiveEndpoints   map[string]*LiveEndpoint `json:"LiveEndpoints,omitempty"`
	LessonStage     int32                    `json:"LessonStage,omitempty"`
	LabGuide        string                   `json:"LabGuide,omitempty"`
	JupyterLabGuide bool                     `json:"JupyterLabGuide,omitempty"`
	Status          string                   `json:"Status,omitempty"`
	CreatedTime     *timestamp.Timestamp     `json:"createdTime,omitempty"`
	Error           bool                     `json:"Error,omitempty"`
	HealthyTests    int32                    `json:"HealthyTests,omitempty"`
	TotalTests      int32                    `json:"TotalTests,omitempty"`
	Busy            bool                     `json:"Busy,omitempty"`
}

// LiveEndpoint is a running instance of a LessonEndpoint, with additional details
// that are relevant at runtime.
type LiveEndpoint struct {
	ID            int32               `json:"ID,omitempty"`
	LiveLessonID  string              `json:"LiveLessonID,omitempty"`
	Name          string              `json:"Name,omitempty"`
	Image         string              `json:"Image,omitempty"`
	Presentations []*LivePresentation `json:"Presentations,omitempty"`
	Host          string              `json:"Host,omitempty"`
}

// LivePresentation is a running instance of a LessonPresentation, with additional details
// that are relevant at runtime.
type LivePresentation struct {
	ID             int32  `json:"ID,omitempty"`
	LiveEndpointID string `json:"LiveEndpointID,omitempty"`
	Name           string `json:"Name,omitempty"`
	Port           int32  `json:"Port,omitempty"`
	Type           string `json:"Type,omitempty"`
}

// Might be worth implementing these fields as well
// type PresentationType int32

// const (
// 	PresentationType_http Status = 1
// 	PresentationType_ssh  Status = 2
// )

type Status int32

const (
	Status_DONOTUSE      Status = 0
	Status_INITIAL_BOOT  Status = 1
	Status_CONFIGURATION Status = 2
	Status_READY         Status = 3
)

var Status_name = map[int32]string{
	0: "DONOTUSE",
	1: "INITIAL_BOOT",
	2: "CONFIGURATION",
	3: "READY",
}

var Status_value = map[string]int32{
	"DONOTUSE":      0,
	"INITIAL_BOOT":  1,
	"CONFIGURATION": 2,
	"READY":         3,
}
