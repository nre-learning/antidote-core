package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	logrus "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"k8s.io/client-go/kubernetes"

	nats "github.com/nats-io/nats.go"
	config "github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
	// Custom Network CRD Types
	// Kubernetes Types
	// Kubernetes clients
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
	Init() error

	handleRequestCREATE(ot.SpanContext, services.LessonScheduleRequest)
	handleRequestMODIFY(ot.SpanContext, services.LessonScheduleRequest)
	handleRequestBOOP(ot.SpanContext, services.LessonScheduleRequest) // TODO - consider if this is necessary anymore. The API has the state, no need to update k8s or other backend labels
	handleRequestDELETE(ot.SpanContext, services.LessonScheduleRequest)

	configureStuff(ot.SpanContext, models.LiveLesson, services.LessonScheduleRequest)
	PurgeOldLessons(ot.SpanContext)
	PurgeOldSessions(ot.SpanContext)
}

// AntidoteScheduler is an Antidote service that receives commands for things like creating new lesson instances,
// moving existing livelessons to a different stage, deleting old lessons, etc.
type AntidoteScheduler struct {
	NC     *nats.Conn
	Config config.AntidoteConfig
	Db     db.DataManager

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
	scheduler.PruneOrphanedNamespaces()

	// Garbage collection
	go func() {
		for {

			span := ot.StartSpan("scheduler_gc")
			// Clean up any old lesson namespaces
			llToDelete, err := s.PurgeOldLessons(span.Context()) // TODO - this is in namespaces.go
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
			err = s.PurgeOldSessions(span.Context())
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
			}

			span.Finish()
			time.Sleep(1 * time.Minute)

		}
	}()

	var handlers = map[services.OperationType]interface{}{
		services.OperationType_CREATE: s.handleRequestCREATE,
		services.OperationType_DELETE: s.handleRequestDELETE,
		services.OperationType_MODIFY: s.handleRequestMODIFY,
		services.OperationType_BOOP:   s.handleRequestBOOP,
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

		go handlers[lsr.Operation].(func(ot.SpanContext, services.LessonScheduleRequest))(span.Context(), lsr)
	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}

// TODO - this should be in the DB package probably?
func (s *AntidoteScheduler) getLiveLessonsForSession(sc ot.SpanContext, lsID string) ([]string, error) {
	span := ot.StartSpan("scheduler_getlivelessonsforsession", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("lsID", lsID)

	llList, err := s.Db.ListLiveLessons(span.Context())
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	retLLIDs := []string{}

	for _, ll := range llList {
		if ll.SessionID == lsID {
			retLLIDs = append(retLLIDs, ll.ID)
		}
	}

	span.LogFields(
		log.Object("llIDs", retLLIDs),
		log.Int("llCount", len(retLLIDs)),
	)

	return retLLIDs, nil
}

// TODO - all of the reachable stuff below should be mandated somehow. The question is, should this be moved into the backend but mandated through interfaces,
// or centralized, and mandate that all of the plugins to use it via documented convention?

func (s *AntidoteScheduler) isEpReachable(ep *models.LiveEndpoint) (bool, error) {

	// rTest is used to give structure to the reachability tests we want to run.
	// This function will first construct a slice of rTests based on information available in the
	// LiveLesson, and then will subsequently run tests based on each rTest.
	type rTest struct {
		name   string
		method string
		host   string
		port   int32
	}

	rTests := []rTest{}
	var mapMutex = &sync.Mutex{}
	reachableMap := map[string]bool{}
	for _, rt := range rTests {
		reachableMap[rt.name] = false
	}

	// If no presentations, add a single rTest using the first available Port.
	// (we aren't currently testing for all Ports)
	if len(ep.Presentations) == 0 {

		if len(ep.Ports) == 0 {
			// Should never happen due to earlier code, but good to be safe
			return false, errors.New("Endpoint has no Ports")
		}

		rTests = append(rTests, rTest{
			name:   ep.Name,
			method: "tcp",
			host:   ep.Host,
			port:   ep.Ports[0],
		})
	}
	for p := range ep.Presentations {
		rTests = append(rTests, rTest{
			name:   fmt.Sprintf("%s-%s", ep.Name, ep.Presentations[p].Name),
			method: string(ep.Presentations[p].Type),
			host:   ep.Host,
			port:   ep.Presentations[p].Port,
		})
	}

	// Last, iterate over the rTests and spawn goroutines for each test
	wg := new(sync.WaitGroup)
	wg.Add(len(rTests))
	for _, rt := range rTests {
		ctx := context.Background()

		// Timeout for an individual test
		ctx, _ = context.WithTimeout(ctx, 10*time.Second)
		go func(ctx context.Context, rt rTest) {
			defer wg.Done()

			testResult := false

			// Not currently doing an HTTP health check, but one could easily be added.
			// rt.method is already being set to "http" for corresponding presentations
			if rt.method == "ssh" {
				testResult = s.HealthChecker.sshTest(rt.host, int(rt.port))
			} else {
				testResult = s.HealthChecker.tcpTest(rt.host, int(rt.port))
			}

			mapMutex.Lock()
			defer mapMutex.Unlock()
			reachableMap[rt.name] = testResult

		}(ctx, rt)
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		for _, r := range reachableMap {
			if !r {
				return false, nil
			}
		}
		return true, nil
	case <-time.After(time.Second * 5):
		return false, nil
	}

}

// waitUntilReachable waits until an entire livelesson is reachable
func (s *AntidoteScheduler) waitUntilReachable(sc ot.SpanContext, ll models.LiveLesson) error {
	span := ot.StartSpan("scheduler_wait_until_reachable", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("liveLessonID", ll.ID)
	span.LogFields(log.Object("liveEndpoints", ll.LiveEndpoints))

	// reachableTimeLimit controls how long we wait for each goroutine to finish
	// as well as in general how long we wait for all of them to finish. If this is exceeded,
	// the livelesson is marked as failed.
	reachableTimeLimit := time.Second * 600

	finishedEps := map[string]bool{}
	wg := new(sync.WaitGroup)
	wg.Add(len(ll.LiveEndpoints))
	for n := range ll.LiveEndpoints {
		ep := ll.LiveEndpoints[n]
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, reachableTimeLimit)

		go func(sc ot.SpanContext, ctx context.Context, ep *models.LiveEndpoint) {
			span := ot.StartSpan("scheduler_ep_reachable_test", ot.ChildOf(sc))
			defer span.Finish()
			span.SetTag("epName", ep.Name)
			span.SetTag("epSSHCreds", fmt.Sprintf("%s:%s", ep.SSHUser, ep.SSHPassword))

			defer wg.Done()
			for {
				epr, err := s.isEpReachable(ep)
				if err != nil {
					span.LogFields(log.Error(err))
					ext.Error.Set(span, true)
					return
				}
				if epr {
					finishedEps[ep.Name] = true
					_ = s.Db.UpdateLiveLessonTests(span.Context(), ll.ID, int32(len(finishedEps)), int32(len(ll.LiveEndpoints)))
					span.LogEvent("Endpoint has become reachable")
					return
				}

				select {
				case <-time.After(1 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
			}
		}(span.Context(), ctx, ep)
	}

	// Wait for each endpoint's goroutine to finish, either through completion,
	// or through context cancelation due to timer expiring.
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		//
	case <-time.After(reachableTimeLimit):
		//
	}

	if len(finishedEps) != len(ll.LiveEndpoints) {
		// Record pod logs for all failed endpoints for later troubleshooting
		for _, ep := range ll.LiveEndpoints {
			if _, ok := finishedEps[ep.Name]; !ok {
				s.recordPodLogs(span.Context(), ll.ID, ep.Name, "")
			}
		}

		err := errors.New("Timeout waiting for LiveEndpoints to become reachable")
		span.LogFields(
			log.Error(err),
			log.Object("failedEps", finishedEps),
		)
		ext.Error.Set(span, true)
		return err
	}

	return nil
}

// usesJupyterLabGuide is a helper function that lets us know if a lesson def uses a
// jupyter notebook as a lab guide in any stage.
func usesJupyterLabGuide(lesson models.Lesson) bool {

	for i := range lesson.Stages {
		if lesson.Stages[i].GuideType == models.GuideJupyter {
			return true
		}
	}

	return false
}

// LessonHealthChecker describes a struct which offers a variety of reachability
// tests for lesson endpoints.
type LessonHealthChecker interface {
	sshTest(string, int) bool
	tcpTest(string, int) bool
}

// LessonHealthCheck performs network tests to determine health of endpoints within a running lesson
type LessonHealthCheck struct{}

// sshTest is an important health check to run especially for interactive endpoints,
// so that we know the endpoint is not only online but ready to receive SSH connections
// from the user via the Web UI
func (lhc *LessonHealthCheck) sshTest(host string, port int) bool {
	strPort := strconv.Itoa(int(port))

	// Using made-up creds, since we only care that SSH is viable for this simple health test.
	sshConfig := &ssh.ClientConfig{
		User:            "foobar",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password("foobar"),
		},
		// TODO(mierdin): This still doesn't seem to work properly for "hung" ssh servers. Having to rely
		// on the outer select/case timeout at the moment.
		Timeout: time.Second * 2,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, strPort), sshConfig)
	if err != nil {
		// For a simple health check, we only care that SSH is responding, not that auth is solid.
		// Thus the made-up creds. If we get this message, then all is good.
		if strings.Contains(err.Error(), "unable to authenticate") {
			return true
		}
		return false
	}
	defer conn.Close()

	return true
}

func (lhc *LessonHealthCheck) tcpTest(host string, port int) bool {
	strPort := strconv.Itoa(int(port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, strPort), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
