package services

import (
	"time"
	// server "github.com/nats-io/nats-server/v2/server"
)

type AntidoteService interface {
	Start() error
}

type OperationType int32

var (
	OperationType_CREATE OperationType = 1
	OperationType_DELETE OperationType = 2
	OperationType_MODIFY OperationType = 3
	OperationType_BOOP   OperationType = 4
)

type LessonScheduleRequest struct {
	Operation     OperationType
	LessonSlug    string
	LiveLessonID  string
	LiveSessionID string
	Stage         int32
	Created       time.Time
}
