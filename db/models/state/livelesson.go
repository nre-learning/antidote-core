package db

import (
	"encoding/json"

	"github.com/golang/protobuf/ptypes/timestamp"
)

func (l *LiveLesson) JSON() string {
	b, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	return string(b)
}

type LiveLesson struct {
	ID               string                   `json:"LiveLessonId,omitempty"`
	SessionID        string                   `json:"SessionId,omitempty"`
	LessonID         int32                    `json:"LessonId,omitempty"`
	LiveEndpoints    map[string]*LiveEndpoint `json:"LessonId,omitempty"`
	LessonStage      int32                    `json:"LessonStage,omitempty"`
	LabGuide         string                   `json:"LabGuide,omitempty"`
	JupyterLabGuide  bool                     `json:"JupyterLabGuide,omitempty"`
	LiveLessonStatus Status                   `json:"LiveLessonStatus,omitempty"`
	CreatedTime      *timestamp.Timestamp     `json:"createdTime,omitempty"`
	Error            bool                     `json:"Error,omitempty"`
	HealthyTests     int32                    `json:"HealthyTests,omitempty"`
	TotalTests       int32                    `json:"TotalTests,omitempty"`
}

type LiveEndpoint struct {
	ID            int32               `json:"ID,omitempty"`
	LiveLessonID  string              `json:"LiveLessonID,omitempty"`
	Name          string              `json:"Name,omitempty"`
	Image         string              `json:"Image,omitempty"`
	Presentations []*LivePresentation `json:"Presentations,omitempty"`
	Host          string              `json:"Host,omitempty"`
}

type LivePresentation struct {
	ID             int32            `json:"Id,omitempty"`
	LiveEndpointID string           `json:"LiveEndpointID,omitempty"`
	Name           string           `json:"Name,omitempty"`
	Port           int32            `json:"Port,omitempty"`
	Type           PresentationType `json:"Type,omitempty"`
}

type Status int32
type PresentationType int32

const (
	Status_INITIAL_BOOT   Status = 1
	Status_CONFIGURATION  Status = 2
	Status_READY          Status = 3
	PresentationType_http Status = 1
	PresentationType_ssh  Status = 2
)
