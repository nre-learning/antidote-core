package scheduler

import (
	"encoding/json"
	"fmt"
	"time"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	logrus "github.com/sirupsen/logrus"

	nats "github.com/nats-io/nats.go"
	config "github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	"github.com/nre-learning/antidote-core/services"
)

// TODO - figure out if this is the right place to insert this abstraction, or if we should do it lower.
// Also the most sticky thing you're likely to experience is the ease of use offered by k8s in deleting a namespace. See if Openstack has a similar property, or punt
// this to the implementor of that plugin
// Need to go through the full deletion and GC process and ensure it's not k8s specific anywhere
//
// This interface will obviously be a requirement, but you'll have opinions on what people do within each fulfillment function. For instance, updating test state as endpoints come online.
// This opens up an interesting question about whether or not those should be defined in an interface. Bottom line, you'll have plugins requirements that go beyond
// implementing these high-level functions, so you should think about those  and document them. What kind of DB functions need to be called, and when?
type SchedulerBackend interface {

	// TODO - this is likely not meant to be implemented in the interface. Each plugin will need an Init function
	// that returns a ready-to-go instance. This will allow for publicly set attributes (antidoteocnfig) but also allow internal types to be set (kubeconfig, etc)
	// Init() error

	PruneOrphans() error
	PruneOldLessons(ot.SpanContext) ([]string, error)
	PruneOldSessions(ot.SpanContext) error

	HandleRequestCREATE(ot.SpanContext, services.LessonScheduleRequest) error
	HandleRequestMODIFY(ot.SpanContext, services.LessonScheduleRequest) error
	HandleRequestBOOP(ot.SpanContext, services.LessonScheduleRequest) error // TODO - consider if this is necessary anymore. The API has the state, no need to update k8s or other backend labels
	HandleRequestDELETE(ot.SpanContext, services.LessonScheduleRequest) error

	// Should these be in the interface? They're called by the create and modify handlers
	// configureStuff(ot.SpanContext, models.LiveLesson, services.LessonScheduleRequest)
	// configureendpoint?
}

// AntidoteScheduler is an Antidote service that receives commands for things like creating new lesson instances,
// moving existing livelessons to a different stage, deleting old lessons, etc.
type AntidoteScheduler struct {
	Config config.AntidoteConfig

	// TODO - NC can probably stay here but Db can be moved to the backend.
	NC *nats.Conn
	Db db.DataManager

	Backend SchedulerBackend

	BuildInfo map[string]string
}

// Start is meant to be run as a goroutine. The "requests" channel will wait for new requests, attempt to schedule them,
// and put a results message on the "results" channel when finished (success or fail)
func (s *AntidoteScheduler) Start() error {

	// TODO - you moved much of the scheduler state to the plugin, so you'll need to copy the logic that the cmd/antidoted/ main is doing, into here based on plugin

	// In case we're restarting from a previous instance, we want to make sure we clean up any
	// orphaned k8s namespaces by killing any with our ID. This should be done synchronously
	// before the scheduler or the API is started.
	logrus.Info("Pruning orphaned namespaces...")
	s.Backend.PruneOrphans()

	// Garbage collection
	go func() {
		for {

			span := ot.StartSpan("scheduler_gc")
			// Clean up any old lesson namespaces
			llToDelete, err := s.Backend.PruneOldLessons(span.Context())
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
			}

			// Clean up local state based on purge results
			for i := range llToDelete {
				err := s.Db.DeleteLiveLesson(span.Context(), llToDelete[i])
				if err != nil {
					span.LogFields(log.Error(err))
					ext.Error.Set(span, true)
				}
			}

			// Clean up old sessions
			err = s.Backend.PruneOldSessions(span.Context())
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
			}

			span.Finish()
			time.Sleep(1 * time.Minute)

		}
	}()

	var handlers = map[services.OperationType]interface{}{
		services.OperationType_CREATE: s.Backend.HandleRequestCREATE,
		services.OperationType_DELETE: s.Backend.HandleRequestDELETE,
		services.OperationType_MODIFY: s.Backend.HandleRequestMODIFY,
		services.OperationType_BOOP:   s.Backend.HandleRequestBOOP,
	}

	s.NC.Subscribe(services.LsrIncoming, func(msg *nats.Msg) {
		t := services.NewTraceMsg(msg)
		tracer := ot.GlobalTracer()
		sc, err := tracer.Extract(ot.Binary, t)
		if err != nil {
			logrus.Errorf("Failed to extract for %s: %v", services.LsrIncoming, err)
		}

		span := ot.StartSpan("scheduler_lsr_incoming", ot.ChildOf(sc))
		defer span.Finish()
		span.LogEvent(fmt.Sprintf("Response msg: %v", msg))

		rem := t.Bytes()
		var lsr services.LessonScheduleRequest
		_ = json.Unmarshal(rem, &lsr)

		go func() {
			err := handlers[lsr.Operation].(func(ot.SpanContext, services.LessonScheduleRequest) error)(span.Context(), lsr)
			if err == nil && lsr.Operation == services.OperationType_CREATE {
				s.NC.Publish(services.LsrCompleted, t.Bytes())
			}
		}()
	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}
