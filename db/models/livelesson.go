package db

import "time"

// LiveLesson is a runtime instance of a Lesson in use. It represents a specific lesson (via LessonID)
// being requested by a specific session (via SessionID) and holds all of the runtime state that Antidote
// needs to know about it to serve it to the front-end appropriately
type LiveLesson struct {
	ID             string                   `json:"LiveLessonId"`
	SessionID      string                   `json:"SessionId"`
	AntidoteID     string                   `json:"AntidoteID"`
	LessonSlug     string                   `json:"LessonSlug"`
	LiveEndpoints  map[string]*LiveEndpoint `json:"LiveEndpoints"`
	CurrentStage   int32                    `json:"LessonStage"`
	GuideContents  string                   `json:"GuideContents"`
	GuideType      string                   `json:"GuideType"`
	GuideDomain    string                   `json:"GuideDomain"`
	Status         LiveLessonStatus         `json:"Status"`
	CreatedTime    time.Time                `json:"CreatedTime"`
	LastActiveTime time.Time                `json:"LastActiveTime"`
	Error          bool                     `json:"Error"`
	HealthyTests   int32                    `json:"HealthyTests"`
	TotalTests     int32                    `json:"TotalTests"`
	Diagram        string                   `json:"Diagram"`
	Video          string                   `json:"Video"`
	StageVideo     string                   `json:"StageVideo"`
}

// LiveEndpoint is a running instance of a LessonEndpoint, with additional details
// that are relevant at runtime.
type LiveEndpoint struct {
	Name              string              `json:"Name"`
	Image             string              `json:"Image"`
	Ports             []int32             `json:"Ports"`
	Presentations     []*LivePresentation `json:"Presentations"`
	ConfigurationType string              `json:"ConfigurationType"`
	ConfigurationFile string              `json:"ConfigurationFile"`
	SSHUser           string              `json:"SSHUser"`
	SSHPassword       string              `json:"SSHPassword"`

	Host string `json:"Host"`
}

// LivePresentation is a running instance of a LessonPresentation, with additional details
// that are relevant at runtime.
type LivePresentation struct {
	Name      string           `json:"Name"`
	Port      int32            `json:"Port"`
	Type      PresentationType `json:"Type"`
	HepDomain string           `json:"HepDomain"`
}

// LiveLessonStatus is backed by a set of possible const values for livelesson statuses below
type LiveLessonStatus string

const (
	// Status_INITIALIZED means the livelesson has been created but not yet executed upon
	// by the scheduler
	Status_INITIALIZED LiveLessonStatus = "INITIALIZED"

	// Status_BOOTING means the necessary instructions for initial resource creation
	// have been/are being sent to Kubernetes and we are waiting for health checks to pass
	Status_BOOTING LiveLessonStatus = "BOOTING"

	// Status_CONFIGURATION means health checks have passed, and we have sent instructions to Kubernetes
	// to spin up configuration pods/jobs, and are waiting for those to finish. This status can be used for
	// both initial configuration, as well as inter-stage configuration (moving from BOOTING to CONFIGURATION
	// or from READY to CONFIGURATION)
	Status_CONFIGURATION LiveLessonStatus = "CONFIGURATION"

	// Status_READY indicates that a livelesson is ready for consumption
	Status_READY LiveLessonStatus = "READY"
)
