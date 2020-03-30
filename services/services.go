package services

// server "github.com/nats-io/nats-server/v2/server"

type AntidoteService interface {
	Start() error
}

type OperationType int32

const (
	OperationType_CREATE OperationType = 1
	OperationType_DELETE OperationType = 2
	OperationType_MODIFY OperationType = 3
	OperationType_BOOP   OperationType = 4

	LsrIncoming  = "antidote.lsr.incoming"
	LsrCompleted = "antidote.lsr.completed"
)

type LessonScheduleRequest struct {
	Operation     OperationType
	LiveLessonID  string
	LiveSessionID string

	// The fields below should eventually be deprecated. Really, all we need in an LSR are IDs for the relevant state
	// (livelessons, livesessions) and the operation that's taking place. All state should be retrieved
	// via lookup with the provided IDs
	//
	// However, it doesn't seem to be causing huge problems at the moment, so this is a low priority.
	LessonSlug string
	Stage      int32
}
