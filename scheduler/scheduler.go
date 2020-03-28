package scheduler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	NEC           *nats.EncodedConn
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

	// TODO(mierdin): Maybe not an issue right now, but should consider if we should check if another Syringe is operating with
	// our configured ID.

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
			span.Finish()
			time.Sleep(1 * time.Minute)

		}
	}()

	// Handle incoming requests asynchronously
	var handlers = map[services.OperationType]interface{}{
		services.OperationType_CREATE: s.handleRequestCREATE,
		services.OperationType_DELETE: s.handleRequestDELETE,
		services.OperationType_MODIFY: s.handleRequestMODIFY,
		services.OperationType_BOOP:   s.handleRequestBOOP,
	}

	// Handling incoming LSRs

	// I **think** that in order to integrate tracing into NATS and get span contexts,
	// we need to use Conn instead of EncodedConn

	// s.NEC.Subscribe("antidote.lsr.incoming", func(lsr services.LessonScheduleRequest) {
	s.NC.Subscribe("antidote.lsr.incoming", func(msg *nats.Msg) {

		// // Create new TraceMsg from the NATS message.
		t := services.NewTraceMsg(msg)

		tracer := ot.GlobalTracer()
		// Extract the span context from the request message.
		sc, err := tracer.Extract(ot.Binary, t)
		if err != nil {
			logrus.Errorf("Failed to extract for antidote.lsr.completed: %v", err)
		}

		span := ot.StartSpan("scheduler_lsr_incoming", ot.ChildOf(sc))
		defer span.Finish()

		span.LogEvent(fmt.Sprintf("Response msg: %v", msg))

		rem := t.Bytes()
		var lsr services.LessonScheduleRequest
		_ = json.Unmarshal(rem, &lsr)

		// Add the current span context to the LSR
		// lsr.SpanContext = span.Context()

		// TODO(mierdin): I **believe** this NATS handler function already runs async, so there's no need to run
		// the below in a goroutine, but should verify this.
		// go func() {
		handlers[lsr.Operation].(func(ot.SpanContext, services.LessonScheduleRequest))(span.Context(), lsr)
		// }()

	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}

func (s *AntidoteScheduler) configureStuff(sc ot.SpanContext, nsName string, ll *models.LiveLesson, newRequest services.LessonScheduleRequest) error {
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

		// TODO(mierdin): This function only sends configuration job requests, should rename accordingly.
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
func (s *AntidoteScheduler) getVolumesConfiguration(sc ot.SpanContext, lessonSlug string) ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container) {
	span := ot.StartSpan("scheduler_get_volumes", ot.ChildOf(sc))
	defer span.Finish()

	lesson, err := s.Db.GetLesson(span.Context(), lessonSlug)
	if err != nil {
		// TODO(mierdin): This function doesn't return an error, which is problematic.
		return nil, nil, nil
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initContainers := []corev1.Container{}

	// Init container will mount the host directory as read-only, and copy entire contents into an emptyDir volume
	initContainers = append(initContainers, corev1.Container{
		Name:  "copy-local-files",
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

	return volumes, volumeMounts, initContainers

}

// waitUntilReachable is the top-level endpoint reachability workflow
func (s *AntidoteScheduler) waitUntilReachable(sc ot.SpanContext, ll *models.LiveLesson) error {
	span := ot.StartSpan("scheduler_wait_until_reachable", ot.ChildOf(sc))
	defer span.Finish()

	span.SetTag("liveLessonID", ll.ID)
	span.LogFields(log.Object("liveEndpoints", ll.LiveEndpoints))

	var success = false
	for i := 0; i < 600; i++ {
		time.Sleep(1 * time.Second)
		reachability := s.getEndpointReachability(span.Context(), ll)
		span.LogFields(log.Object("reachability", reachability))

		// Update reachability status
		failed := false
		healthy := int32(0)
		total := int32(len(reachability))
		for _, reachable := range reachability {
			if reachable {
				healthy++
			} else {
				failed = true
			}
		}
		ll.HealthyTests = healthy
		ll.TotalTests = total

		// Begin again if one of the endpoints isn't reachable
		if failed {
			continue
		}

		success = true
		break

	}

	if !success {
		err := errors.New("Timeout waiting for LiveEndpoints to become reachable")
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return err
	}

	return nil
}

// getEndpointReachability does a one-time health check for network reachability for endpoints in a
// livelesson. Meant to be called by a higher-level function which calls it repeatedly until all
// are healthy.
func (s *AntidoteScheduler) getEndpointReachability(sc ot.SpanContext, ll *models.LiveLesson) map[string]bool {

	span := ot.StartSpan("scheduler_get_endpoint_reachability", ot.ChildOf(sc))
	defer span.Finish()

	// rTest is used to give structure to the reachability tests we want to run.
	// This function will first construct a slice of rTests based on information available in the
	// LiveLesson, and then will subsequently run tests based on each rTest.
	type rTest struct {

		// name should have
		name   string
		method string
		host   string
		port   int32
	}

	// First construct a slice of rTests. This allows us to first get a sense for what
	//we're about to test and how
	rTests := []rTest{}
	for n := range ll.LiveEndpoints {
		ep := ll.LiveEndpoints[n]

		// If no presentations, add a single rTest using the first available Port.
		// (we aren't currently testing for all Ports)
		if len(ll.LiveEndpoints[n].Presentations) == 0 {
			rTests = append(rTests, rTest{
				name:   ep.Name,
				method: "tcp",
				host:   ep.Host,
				port:   ep.Ports[0], // TODO(mierdin): SHOULD be present due to earlier code, but to be safe, may want to add a check
			})
			continue
		}
		for p := range ep.Presentations {
			rTests = append(rTests, rTest{
				name:   fmt.Sprintf("%s-%s", ep.Name, ep.Presentations[p].Name),
				method: string(ep.Presentations[p].Type),
				host:   ep.Host,
				port:   ep.Presentations[p].Port,
			})
		}
	}

	span.LogFields(log.Object("rTests", rTests))

	// Next, we need to seed the map we'll be returning from this function with all of the tests
	// we expect a status on, set to false by default. This is important, as the higher-level code will use this to determine
	// when the test is finished (all must be true)
	var mapMutex = &sync.Mutex{}
	reachableMap := map[string]bool{}
	for _, rt := range rTests {
		reachableMap[rt.name] = false
	}

	// Last, iterate over the rTests and spawn goroutines for each test
	wg := new(sync.WaitGroup)
	wg.Add(len(rTests))
	for _, rt := range rTests {
		go func(span ot.Span, rt rTest) {
			defer wg.Done()

			testResult := false

			// TODO(mierdin): Consider adding an HTTP health check, but not urgently needed atm
			if rt.method == "ssh" {
				testResult = s.HealthChecker.sshTest(rt.host, int(rt.port))
			} else {
				testResult = s.HealthChecker.tcpTest(rt.host, int(rt.port))
			}
			span.LogFields(
				log.String("rtName", rt.name),
				log.String("rtMethod", rt.method),
				log.String("testTarget", fmt.Sprintf("%s:%d", rt.host, rt.port)),
				log.Bool("testResult", testResult),
			)

			mapMutex.Lock()
			defer mapMutex.Unlock()
			reachableMap[rt.name] = testResult

		}(span, rt)
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		return reachableMap
	case <-time.After(time.Second * 10):
		return reachableMap
	}
}

func (s *AntidoteScheduler) getPodLogs(pod *corev1.Pod) string {

	req := s.Client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	podLogs, err := req.Stream()
	if err != nil {
		return "error in opening stream"
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf"
	}
	str := buf.String()

	return str
}

// usesJupyterLabGuide is a helper function that lets us know if a lesson def uses a
// jupyter notebook as a lab guide in any stage.
func usesJupyterLabGuide(lesson *models.Lesson) bool {

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
