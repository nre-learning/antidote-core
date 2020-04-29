package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	logrus "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	nats "github.com/nats-io/nats.go"
	config "github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"

	// Custom Network CRD Types
	networkcrd "github.com/nre-learning/antidote-core/pkg/apis/k8s.cni.cncf.io/v1"

	// Kubernetes Types
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rest "k8s.io/client-go/rest"

	// Kubernetes clients

	kubernetesCrd "github.com/nre-learning/antidote-core/pkg/client/clientset/versioned"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
)

const initContainerName string = "copy-local-files"

// NetworkCrdClient is an interface for the client for our custom
// network CRD. Allows for injection of mocks at test time.
type NetworkCrdClient interface {
	UpdateNamespace(string)
	Create(obj *networkcrd.NetworkAttachmentDefinition) (*networkcrd.NetworkAttachmentDefinition, error)
	Update(obj *networkcrd.NetworkAttachmentDefinition) (*networkcrd.NetworkAttachmentDefinition, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	Get(name string) (*networkcrd.NetworkAttachmentDefinition, error)
	List(opts meta_v1.ListOptions) (*networkcrd.NetworkList, error)
}

// AntidoteScheduler is an Antidote service that receives commands for things like creating new lesson instances,
// moving existing livelessons to a different stage, deleting old lessons, etc.
type AntidoteScheduler struct {
	KubeConfig    *rest.Config
	NC            *nats.Conn
	Config        config.AntidoteConfig
	Db            db.DataManager
	HealthChecker LessonHealthChecker

	// Allows us to disable GC for testing. Production code should leave this at
	// false
	// DisableGC bool

	// Client for interacting with normal Kubernetes resources
	Client kubernetes.Interface

	// Client for creating CRD defintions
	ClientExt kubernetesExt.Interface

	// Client for creating instances of our network CRD
	ClientCrd kubernetesCrd.Interface

	BuildInfo map[string]string
}

// Start is meant to be run as a goroutine. The "requests" channel will wait for new requests, attempt to schedule them,
// and put a results message on the "results" channel when finished (success or fail)
func (s *AntidoteScheduler) Start() error {

	// Ensure our network CRD is in place (should fail silently if already exists)
	err := s.createNetworkCrd()
	if err != nil {
		return err
	}

	// Garbage collection
	go func() {
		for {

			span := ot.StartSpan("scheduler_gc")
			// Clean up any old lesson namespaces
			llToDelete, err := s.PurgeOldLessons(span.Context())
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

// PurgeOldSessions cleans up old LiveSessions according to the configured LiveSessionTTL
func (s *AntidoteScheduler) PurgeOldSessions(sc ot.SpanContext) error {
	span := ot.StartSpan("scheduler_purgeoldsessions", ot.ChildOf(sc))
	defer span.Finish()

	lsList, err := s.Db.ListLiveSessions(span.Context())
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	lsTTL := time.Duration(s.Config.LiveSessionTTL) * time.Minute

	for _, ls := range lsList {
		createdTime := time.Since(ls.CreatedTime)

		// No need to continue if this session hasn't even exceeded the TTL
		if createdTime <= lsTTL {
			continue
		}

		llforls, err := s.getLiveLessonsForSession(span.Context(), ls.ID)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}

		// We don't want/need to clean up this session if there are active livelessons that are using it.
		if len(llforls) > 0 {
			continue
		}

		// TODO(mierdin): It would be pretty rare, but in the event that a livelesson is spun up between the request above
		// and the livesession deletion below, we would encounter the leak bug we saw in 0.6.0. It might be worth seeing if
		// you can lock things somehow between the two.

		err = s.Db.DeleteLiveSession(span.Context(), ls.ID)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}
	}

	return nil
}

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

func (s *AntidoteScheduler) configureStuff(sc ot.SpanContext, nsName string, ll models.LiveLesson, newRequest services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_configure_stuff", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", ll.ID)
	span.LogFields(log.Object("llEndpoints", ll.LiveEndpoints))

	s.killAllJobs(span.Context(), nsName, "config")

	wg := new(sync.WaitGroup)
	wg.Add(len(ll.LiveEndpoints))
	allGood := true
	for i := range ll.LiveEndpoints {

		// Ignore any endpoints that don't have a configuration option
		if ll.LiveEndpoints[i].ConfigurationType == "" {
			wg.Done()
			continue
		}

		job, err := s.configureEndpoint(span.Context(), ll.LiveEndpoints[i], newRequest)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}

		go func() {
			defer wg.Done()

			oldStatusCount := map[string]int32{
				"active":    0,
				"succeeded": 0,
				"failed":    0,
			}
			for i := 0; i < 600; i++ {
				completed, statusCount, err := s.getJobStatus(span, job, newRequest)
				if err != nil {
					allGood = false
					return
				}

				// The use of this map[string]int32 and comparing old with new using DeepEqual
				// allows us to only log changes in status, rather than the periodic spam
				if !reflect.DeepEqual(oldStatusCount, statusCount) {
					span.LogFields(
						log.String("jobName", job.Name),
						log.Object("statusCount", statusCount),
					)
				}

				if completed {
					return
				}
				oldStatusCount = statusCount
				time.Sleep(1 * time.Second)
			}
			allGood = false
			return
		}()

	}

	wg.Wait()

	if !allGood {
		return errors.New("Problem during configuration")
	}

	return nil
}

// getVolumesConfiguration returns a slice of Volumes, VolumeMounts, and init containers that should be used in all pod and job definitions.
// This allows Syringe to pull lesson data from either Git, or from a local filesystem - the latter of which being very useful for lesson
// development.
func (s *AntidoteScheduler) getVolumesConfiguration(sc ot.SpanContext, lessonSlug string) ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container, error) {
	span := ot.StartSpan("scheduler_get_volumes", ot.ChildOf(sc))
	defer span.Finish()

	lesson, err := s.Db.GetLesson(span.Context(), lessonSlug)
	if err != nil {
		return nil, nil, nil, err
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initContainers := []corev1.Container{}

	// Init container will mount the host directory as read-only, and copy entire contents into an emptyDir volume
	initContainers = append(initContainers, corev1.Container{
		Name:  initContainerName,
		Image: "bash",
		Command: []string{
			"bash",
		},
		Args: []string{
			"-c",
			"cp -r /antidote-ro/lessons/ /antidote && adduser -D antidote && chown -R antidote:antidote /antidote",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "host-volume",
				ReadOnly:  true,
				MountPath: "/antidote-ro",
			},
			{
				Name:      "local-copy",
				ReadOnly:  false,
				MountPath: "/antidote",
			},
		},
	})

	// Add outer host volume, should be mounted read-only
	volumes = append(volumes, corev1.Volume{
		Name: "host-volume",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: s.Config.CurriculumDir,
			},
		},
	})

	// Add inner container volume, should be mounted read-write so we can copy files into it
	volumes = append(volumes, corev1.Volume{
		Name: "local-copy",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// Finally, mount local copy volume as read-write
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "local-copy",
		ReadOnly:  false,
		MountPath: "/antidote",
		SubPath:   strings.TrimPrefix(lesson.LessonDir, fmt.Sprintf("%s/", path.Clean(s.Config.CurriculumDir))),
	})

	return volumes, volumeMounts, initContainers, nil

}

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
