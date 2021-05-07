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
	"github.com/nre-learning/antidote-core/services"
)

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

	// Deletion of back-end infrastructure should be done with the fewest steps possible. See the note above HandleRequestDELETE for more details.
	PruneOrphans() error

	// Responsible for cleaning up infrastructure resources AND cleaning up state
	// Deletion of back-end infrastructure should be done with the fewest steps possible. See the note above HandleRequestDELETE for more details.
	PruneOldLiveLessons(ot.SpanContext) error

	// PruneOldLiveSessions cleans up livesessions that no longer have active livelessons and have lived past the livesession TTL. No back-end infrastructure
	// TODO - this can/should be done at the scheduler level, since this has no backend requirements
	PruneOldLiveSessions(ot.SpanContext) error

	// HandleRequestCREATE handles a livelesson creation request.
	// Functions that satisfy this portion of the interface must also follow some specific implementation details, which
	// are documented below.
	//
	// 	1. Retrieve lesson and livelesson details from the DataManager, using the incoming livelesson ID
	//  2. Based on these details, provision all necessary infrastructure with the back-end provider. This is an important step, so there are some sub-points to
	//     be made here:
	//     - ALL possible features available in a lesson definition must be supported - all lesson guide types, endpoint presentation types, connections, etc.
	//     - All endpoints must be reachable via the port(s) listed in the lesson definition, either directly, or via some kind of L7 proxy where relevant.
	//     - Care should be taken to "decorate" all created infrastructure resources so that it can be easily looked up later, using some kind of label.
	//       This becomes extremely important when cleaning up old/unused lesson resources. At a minimum, the Antidote instance ID, the corresponding
	//       livelesson ID, and the corresponding livesession ID should be included. Note also that some infrastructure providers offer hierarchical
	//       organization types (e.g. "namespaces" in kubernetes) under which all other resources can be created. This should be employed whenever available, and labels
	//       should be applied to all created resources.
	//     - For created resources that cannot be namespaced, the unique identifier for these resources must be disambiguated using a combination of the livelesson ID and
	//       the Antidote instance ID.
	//  3. Continue to poll the backend infrastructure provider until IP addresses are provisioned for each endpoint. These IP addresses must be reachable from the
	//     antidoted service, as well as any external services that will connect to those endpoints, such as the WebSSH proxy. Once known, the UpdateLiveLessonEndpointIP
	//     DataManager function should be used to update the livelesson details with this IP address.
	//  4. If at any point a failure occurs, capture as much detail as possible and export into the OpenTracing span for this function. This could include logs from endpoints,
	//     or from the backend provider itself.
	// 	5. Use the `WaitUntilReachable` function in the `reachability` package to initiate reachability testing for all endpoints in this livelesson. This function
	//     will take care of updating endpoint test state as they come online, so you need only block execution while waiting for this function to return, and handle
	//     a non-nil error value by passing it up the stack.
	// 	6. If the `WaitUntilReachable` function returns no error, the livelesson should be updated with a status of `CONFIGURATION`.
	// 	7. All endpoints with a configuration option defined in the lesson definition should be configured accordingly. All possible configuration options must be supported.
	//     The `antidote-images` repository contains dockerfiles and scripts that may be useful. Configuration should be entirely atomic - configuration of a given stage
	//     cannot rely on another stage being configured first.
	// 	8. Sleep for the number of seconds indicated by the ReadyDelay configuration option.
	// 	9. Update livelesson status to `READY`. At this point, this function can return a nil error, provided this update took place successfully.
	HandleRequestCREATE(ot.SpanContext, services.LessonScheduleRequest) error

	// HandleRequestMODIFY handles a request to modify an existing livelesson, typically in response to a change to a different stage.
	// The API typically handles some initial state management, but the scheduler backend must execute any endpoint reconfigurations specified by the lesson.
	//
	// This function should perform the exact same configuration logic, and adhere to the same requirements of, the configuration step during the creation process. For this
	// reason, it might be best to contain this logic in an isolated function that can be called from HandleRequestCREATE and HandleRequestMODIFY.
	HandleRequestMODIFY(ot.SpanContext, services.LessonScheduleRequest) error

	// HandleRequestDELETE handles a request to delete an existing livelesson. This is typically done as part of the internal scheduler GC process - not invoked
	// by an end-user.
	//
	// This should rely heavily on metadata provided to the back-end infrastructure provider on creation, so that ONLY the infrastructure for the
	// specific livelesson and antidote instance can be deleted. In addition, **whenever possible**, the back-end infrastructure provider
	// should do the heavy lifting here. For example, the Kubernetes backend creates everything in HandleRequestCREATE within a parent namespace, so that
	// when a livelesson's resources need to be cleaned up, a single API request to Kubernetes deletes **everything**, and a failure in the Antidote service
	// doesn't impact this process. This approach should be followed if at all possible, as opposed to going through and deleting every individual resource you created in
	// HandleRequestCREATE.
	HandleRequestDELETE(ot.SpanContext, services.LessonScheduleRequest) error
}

// AntidoteScheduler handles the high-level orchestration of backend tasks, including delegation of incoming events to
// the active backend implementation and garbage collection of old/unused lesson resources.
type AntidoteScheduler struct {
	Config    config.AntidoteConfig
	NC        *nats.Conn
	Backend   AntidoteBackend
	BuildInfo map[string]string
}

// Start is meant to be run as a goroutine. The "requests" channel will wait for new requests, attempt to schedule them,
// and put a results message on the "results" channel when finished (success or fail)
func (s *AntidoteScheduler) Start() error {

	// Important that this is done first, synchronously, prior to starting GC or listening for LSRs
	logrus.Info("Pruning orphaned lesson resources...")
	s.Backend.PruneOrphans()

	// Garbage collection
	go func() {
		for {
			span := ot.StartSpan("scheduler_gc")
			err := s.Backend.PruneOldLiveLessons(span.Context())
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
			}

			err = s.Backend.PruneOldLiveSessions(span.Context())
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
