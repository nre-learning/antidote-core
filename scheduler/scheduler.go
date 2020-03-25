package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
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
	s.createNetworkCrd()

	// TODO(mierdin): Maybe not an issue right now, but should consider if we should check if another Syringe is operating with
	// our configured ID.

	// Garbage collection
	go func() {
		for {

			span := opentracing.StartSpan("scheduler_gc")
			// Clean up any old lesson namespaces
			llToDelete, err := s.PurgeOldLessons(span.Context())
			if err != nil {
				log.Error("Problem with GCing lessons")
			}

			// Clean up local state based on purge results
			for i := range llToDelete {
				err := s.Db.DeleteLiveLesson(span.Context(), llToDelete[i])
				if err != nil {
					log.Errorf("Unable to delete livelesson %s after GC: %v", llToDelete[i], err)
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

		tracer := opentracing.GlobalTracer()
		// Extract the span context from the request message.
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			log.Printf("Extract error: %v", err)
		}

		span := tracer.StartSpan(
			"scheduler_lsr_incoming",
			opentracing.ChildOf(sc))
		defer span.Finish()

		span.LogEvent(fmt.Sprintf("Response msg: %v", msg))

		rem := t.Bytes()
		var lsr services.LessonScheduleRequest
		_ = json.Unmarshal(rem, &lsr)

		// Add the current span context to the LSR
		// lsr.SpanContext = span.Context()

		// TODO(mierdin): I **believe** this function already runs async, so there's no need to run
		// the below in a goroutine, but should verify this.
		// go func() {
		handlers[lsr.Operation].(func(opentracing.SpanContext, services.LessonScheduleRequest))(span.Context(), lsr)
		// }()

	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}

func (s *AntidoteScheduler) configureStuff(sc opentracing.SpanContext, nsName string, liveLesson *models.LiveLesson, newRequest services.LessonScheduleRequest) error {
	span := opentracing.StartSpan("scheduler_configure_stuff", opentracing.ChildOf(sc))
	defer span.Finish()

	s.killAllJobs(nsName, "config")

	wg := new(sync.WaitGroup)
	log.Debugf("Endpoints length: %d", len(liveLesson.LiveEndpoints))
	wg.Add(len(liveLesson.LiveEndpoints))
	allGood := true
	for i := range liveLesson.LiveEndpoints {

		// Ignore any endpoints that don't have a configuration option
		if liveLesson.LiveEndpoints[i].ConfigurationType == "" {
			log.Debugf("No configuration option specified for %s - skipping.", liveLesson.LiveEndpoints[i].Name)
			wg.Done()
			continue
		}

		job, err := s.configureEndpoint(span.Context(), liveLesson.LiveEndpoints[i], newRequest)
		if err != nil {
			log.Errorf("Problem configuring endpoint %s", liveLesson.LiveEndpoints[i].Name)
			continue // TODO(mierdin): should quit entirely and return an error result to the channel
			// though this error is only immediate errors creating the job. This will succeed even if
			// the eventually configuration fais. See below for a better handle on configuration failures.
		}
		go func() {
			defer wg.Done()

			for i := 0; i < 600; i++ {
				completed, err := s.isCompleted(job, newRequest)
				if err != nil {
					allGood = false
					return
				}
				if completed {
					return
				}
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
func (s *AntidoteScheduler) getVolumesConfiguration(sc opentracing.SpanContext, lessonSlug string) ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container) {
	span := opentracing.StartSpan("scheduler_get_volumes", opentracing.ChildOf(sc))
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
func (s *AntidoteScheduler) waitUntilReachable(sc opentracing.SpanContext, ll *models.LiveLesson) error {
	span := opentracing.StartSpan("scheduler_endpoint_reachability", opentracing.ChildOf(sc))
	defer span.Finish()

	var success = false
	for i := 0; i < 600; i++ {
		time.Sleep(1 * time.Second)

		log.Debugf("About to test endpoint reachability for livelesson %s with endpoints %v", ll.ID, ll.LiveEndpoints)

		reachability := s.getEndpointReachability(ll)

		log.Debugf("Livelesson %s health check results: %v", ll.ID, reachability)

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
		log.Errorf("Timeout waiting for livelesson %s to become reachable", ll.ID)
		err := s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		if err != nil {
			log.Errorf("Error updating livelesson: %v", err)
		}
		return errors.New("Problem")
	}

	return nil
}

// getEndpointReachability does a one-time health check for network reachability for endpoints in a
// livelesson. Meant to be called by a higher-level function which calls it repeatedly until all
// are healthy.
func (s *AntidoteScheduler) getEndpointReachability(ll *models.LiveLesson) map[string]bool {

	// Prepare the reachability map as well as the waitgroup to handle the concurrency
	// of our health checks. We want to pre-populate every possible health check with a
	// false value, so we don't accidentally "pass" a livelesson reachability test by
	// omission.
	reachableMap := map[string]bool{}
	wg := new(sync.WaitGroup)
	for n := range ll.LiveEndpoints {

		ep := ll.LiveEndpoints[n]

		// Add one delta value to the waitgroup and prepopulate the reachability map
		// with a "false" value based on the endpoint name, since it doesn't
		// have any presentations.
		if len(ll.LiveEndpoints[n].Presentations) == 0 {
			wg.Add(1)
			reachableMap[ep.Name] = false
			continue
		}

		// For each presentation, add one delta value to the waitgroup
		// and add an entry to the reachability map based on the endpoint
		// and presentation names
		for p := range ep.Presentations {
			wg.Add(1)
			reachableMap[fmt.Sprintf("%s-%s", ep.Name, ep.Presentations[p].Name)] = false
		}
	}

	// Now that we have a properly sized waitgroup and a prepared reachability map, we can perform the health checks.
	var mapMutex = &sync.Mutex{}
	for n := range ll.LiveEndpoints {

		ep := ll.LiveEndpoints[n]

		// TODO(mierdin): Since you're collapsing all ports into the LiveEndpoint "Ports" property,
		// You may be able to simplify below

		// If no presentations, we can just test the first port in the additionalPorts list.
		if len(ep.Presentations) == 0 && len(ep.Ports) != 0 {

			go func() {
				defer wg.Done()
				testResult := false

				log.Debugf("Performing basic connectivity test against endpoint %s via %s:%d", ep.Name, ep.Host, ep.Ports[0])
				testResult = s.HealthChecker.tcpTest(ep.Host, int(ep.Ports[0]))

				if testResult {
					log.Debugf("%s is live at %s:%d", ep.Name, ep.Host, ep.Ports[0])
				}

				mapMutex.Lock()
				defer mapMutex.Unlock()
				reachableMap[ep.Name] = testResult
			}()
		}

		for i := range ep.Presentations {

			lp := ep.Presentations[i]

			go func() {
				defer wg.Done()

				testResult := false

				// TODO(mierdin): Switching to TCP testing for all endpoints for now. The SSH health check doesn't seem to respect the
				// timeout settings I'm passing, and the regular TCP test does, so I'm using that for now. It's good enough for the time being.
				log.Debugf("Performing basic connectivity test against %s-%s via %s:%d", ep.Name, lp.Name, ep.Host, lp.Port)
				testResult = s.HealthChecker.tcpTest(ep.Host, int(lp.Port))
				if testResult {
					log.Debugf("%s-%s is live at %s:%d", ep.Name, lp.Name, ep.Host, lp.Port)
				}

				mapMutex.Lock()
				defer mapMutex.Unlock()
				reachableMap[fmt.Sprintf("%s-%s", ep.Name, lp.Name)] = testResult

			}()
		}
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

func (lhc *LessonHealthCheck) sshTest(host string, port int) bool {
	strPort := strconv.Itoa(int(port))
	sshConfig := &ssh.ClientConfig{
		User:            "antidote",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password("antidotepassword"),
		},
		Timeout: time.Second * 2,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, strPort), sshConfig)
	if err != nil {
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
