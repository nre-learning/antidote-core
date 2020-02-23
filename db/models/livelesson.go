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
	Status          LiveLessonStatus         `json:"Status,omitempty"`
	CreatedTime     *timestamp.Timestamp     `json:"createdTime,omitempty"`
	Error           bool                     `json:"Error,omitempty"`
	HealthyTests    int32                    `json:"HealthyTests,omitempty"`
	TotalTests      int32                    `json:"TotalTests,omitempty"`
	Busy            bool                     `json:"Busy,omitempty"`
}

// LiveEndpoint is a running instance of a LessonEndpoint, with additional details
// that are relevant at runtime.
type LiveEndpoint struct {
	ID                int32               `json:"ID,omitempty"`
	LiveLessonID      string              `json:"LiveLessonID,omitempty"`
	Name              string              `json:"Name,omitempty"`
	Image             string              `json:"Image,omitempty"`
	Ports             string              `json:"Ports,omitempty"`
	Presentations     []*LivePresentation `json:"Presentations,omitempty"`
	ConfigurationType string              `json:"ConfigurationType,omitempty"`

	Host string `json:"Host,omitempty"`
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

type LiveLessonStatus string

const (
	Status_INITIALIZED   LiveLessonStatus = "INITIALIZED"
	Status_BOOTING       LiveLessonStatus = "BOOTING"
	Status_CONFIGURATION LiveLessonStatus = "CONFIGURATION"
	Status_READY         LiveLessonStatus = "READY"
)
