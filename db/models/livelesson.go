package db

import "time"

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
	ID              string                   `json:"LiveLessonId"`
	SessionID       string                   `json:"SessionId"`
	LessonSlug      string                   `json:"LessonSlug"`
	LiveEndpoints   map[string]*LiveEndpoint `json:"LiveEndpoints"`
	LessonStage     int32                    `json:"LessonStage"`
	LabGuide        string                   `json:"LabGuide"`
	JupyterLabGuide bool                     `json:"JupyterLabGuide"`
	Status          LiveLessonStatus         `json:"Status"`
	CreatedTime     time.Time                `json:"CreatedTime"`
	Error           bool                     `json:"Error"`
	HealthyTests    int32                    `json:"HealthyTests"`
	TotalTests      int32                    `json:"TotalTests"`
	LessonDiagram   string                   `json:"LessonDiagram"`
	LessonVideo     string                   `json:"LessonVideo"`
	Busy            bool                     `json:"Busy"`
}

// LiveEndpoint is a running instance of a LessonEndpoint, with additional details
// that are relevant at runtime.
type LiveEndpoint struct {
	Name              string              `json:"Name"`
	Image             string              `json:"Image"`
	Ports             []int32             `json:"Ports"`
	Presentations     []*LivePresentation `json:"Presentations"`
	ConfigurationType string              `json:"ConfigurationType"`

	Host string `json:"Host"`
}

// LivePresentation is a running instance of a LessonPresentation, with additional details
// that are relevant at runtime.
type LivePresentation struct {
	Name string           `json:"Name"`
	Port int32            `json:"Port"`
	Type PresentationType `json:"Type"`
}

type LiveLessonStatus string

const (
	Status_INITIALIZED   LiveLessonStatus = "INITIALIZED"
	Status_BOOTING       LiveLessonStatus = "BOOTING"
	Status_CONFIGURATION LiveLessonStatus = "CONFIGURATION"
	Status_READY         LiveLessonStatus = "READY"
)
