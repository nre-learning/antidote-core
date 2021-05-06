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

// AntidoteBackend
//
// All methods in this interface return an error type. If a backend provider encounters an unrecoverable error, this can be returned so
// the scheduler is able to notify the front-end of an issue. Before doing this, the backend provider should update the livelesson status to ERROR, set the OpenTracing span context to an error state, and log the error to the span fields. For example:
//
// ```go
// span.LogFields(log.Error(err))
// ext.Error.Set(span, true)
// _ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
// return err
// ```
//
// See the docstring above each method for further implementation detail requirements.
type AntidoteBackend interface {

	// TODO - this is likely not meant to be implemented in the interface. Each plugin will need an Init function
	// that returns a ready-to-go instance. This will allow for publicly set attributes (antidoteocnfig) but also allow internal types to be set (kubeconfig, etc)
	// Init() error

	PruneOrphans() error
	PruneOldLessons(ot.SpanContext) ([]string, error)
	PruneOldSessions(ot.SpanContext) error

	// HandleRequestCREATE handles a livelesson creation request.
	// Functions that satisfy this portion of the interface must also follow some specific implementation details, which
	// are documented below.
	//
	// 	1. Retrieve lesson and livelesson details from the DataManager, using the incoming livelesson ID
	//  2. Based on these details, provision all necessary infrastructure with the back-end provider. ALL possible features available in a lesson definition
	//     must be supported - all lesson guide types, endpoint presentation types, connections, etc. All endpoints must be reachable via the port(s) listed in
	//     the lesson definition, either directly, or via some kind of L7 proxy where relevant. If the backend infrastructure allows, it might be useful to create some
	//     kind of higher-order container or label to place these infrastructure components in, such as a "namespace" in Kubernetes. This makes it a LOT easier to clean
	//     these up during the deletion process.

	// So I don't think BOOP/timestamp is needed, but some kind of label is DEFINITELY needed on all infrastructure elements, so that pruneorphans can work. Unlike the other
	// pruning functions, this one cannot rely on previous state, so we need to be able to search the back-end infrastructure for some kind of label. In this case, timestamp
	// won't matter. Add a quick note here abotu the need for this label and refer to the docstring of pruneorphans.

	// The other two functions SHOULD be able to rely on the timestamp available in livelesson state, (make sure this is getting updated) and therefore the scheduler will only
	// need to be invoked when there's a deletion event. Verify this line of thinking and refactor the scheduler interface/implementation accordingly.

	//  3. Continue to poll the backend infrastructure provider until IP addresses are provisioned for each endpoint. These IP addresses must be reachable from the
	//     antidoted service, as well as any external services that will connect to those endpoints, such as the WebSSH proxy. Once known, the UpdateLiveLessonEndpointIP
	//     DataManager function should be used to update the livelesson details with this IP address.
	//  4. If at any point a failure occurs, capture as much detail as possible and export into the OpenTracing span for this function. This could include logs from endpoints,
	//     or from the backend provider itself.
	// 	5. Use the `WaitUntilReachable` function in the `reachability` package to initiate reachability testing for all endpoints in this livelesson. You should block while
	//     this function executes, and pass any errors up the stack.
	// 	6. If the `WaitUntilReachable` function returns no error, the livelesson should be updated with a status of `CONFIGURATION`.
	// 	7. All endpoints with a configuration option defined in the lesson definition should be configured accordingly. All possible configuration options must be supported.
	//     The `antidote-images` repository contains dockerfiles and scripts that may be useful.
	// 	8. Sleep for the number of seconds indicated by the ReadyDelay configuration option.
	// 	9. Update livelesson status to `READY`. At this point, this function can return a nil error, provided this update took place successfully.
	HandleRequestCREATE(ot.SpanContext, services.LessonScheduleRequest) error

	// HandleRequestMODIFY handles a request to modify an existing livelesson, typically in response to a change to a different stage.
	// The API typically handles some initial state management, but the scheduler backend must execute any endpoint reconfigurations specified by the lesson.
	//
	// Functions that satisfy this portion of the interface must also follow some specific implementation details, which
	// are documented below.
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

	Backend AntidoteBackend

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

				// Inject span context and send LSR into NATS
				tracer := ot.GlobalTracer()
				var t services.TraceMsg
				if err := tracer.Inject(span.Context(), ot.Binary, &t); err != nil {
					span.LogFields(log.Error(err))
					ext.Error.Set(span, true)
				}
				reqBytes, _ := json.Marshal(lsr)
				t.Write(reqBytes)

				s.NC.Publish(services.LsrCompleted, t.Bytes())
			}
		}()
	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}
