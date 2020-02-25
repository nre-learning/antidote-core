package services

import (
	"sync"
	"time"
)

type AntidoteService interface {
	Start() error
}

// This is a simple pub/sub mechanism borrowed from https://levelup.gitconnected.com/lets-write-a-simple-event-bus-in-go-79b9480d8997
// and modified to fit our needs. In our case, we just need some basic event-driven logic in place so we can
// allow our internal services to run asynchronusly and loosely coupled, as opposed to the tightly coupled
// paradigm of the Syringe days.

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

// RequestChannel is a channel which can accept an LessonScheduleRequest
type RequestChannel chan LessonScheduleRequest

// RequestChannelSlice is a slice of RequestChannel
type RequestChannelSlice []RequestChannel

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: map[string]RequestChannelSlice{},
	}
}

// EventBus stores the information about subscribers interested for a particular topic
type EventBus struct {
	subscribers map[string]RequestChannelSlice
	rm          sync.RWMutex
}

func (eb *EventBus) Publish(topic string, req LessonScheduleRequest) {
	eb.rm.RLock()
	if chans, found := eb.subscribers[topic]; found {
		// this is done because the slices refer to same array even though they are passed by value
		// thus we are creating a new slice with our elements thus preserve locking correctly.
		// special thanks for /u/freesid who pointed it out
		channels := append(RequestChannelSlice{}, chans...)
		go func(req LessonScheduleRequest, requestChannelSlices RequestChannelSlice) {
			for _, ch := range requestChannelSlices {
				ch <- req
			}
		}(req, channels)
	}
	eb.rm.RUnlock()
}

func (eb *EventBus) Subscribe(topic string, ch RequestChannel) {
	eb.rm.Lock()
	if prev, found := eb.subscribers[topic]; found {
		eb.subscribers[topic] = append(prev, ch)
	} else {
		eb.subscribers[topic] = append([]RequestChannel{}, ch)
	}
	eb.rm.Unlock()
}
