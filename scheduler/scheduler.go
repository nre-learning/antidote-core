// Responsible for creating all resources for a lab. Pods, services, networks, etc.
package scheduler

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"

	// Custom Network CRD Types
	networkcrd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"

	// Kubernetes Types
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rest "k8s.io/client-go/rest"

	// Kubernetes clients
	kubernetesCrd "github.com/nre-learning/syringe/pkg/client/clientset/versioned"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
)

type OperationType int32

var (
	OperationType_CREATE OperationType = 1
	OperationType_DELETE OperationType = 2
	OperationType_MODIFY OperationType = 3
	OperationType_BOOP   OperationType = 4
	OperationType_VERIFY OperationType = 5
	defaultGitFileMode   int32         = 0755
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

type LessonScheduler struct {
	KubeConfig    *rest.Config
	Requests      chan *LessonScheduleRequest
	Results       chan *LessonScheduleResult
	Curriculum    *pb.Curriculum
	SyringeConfig *config.SyringeConfig
	GcWhiteList   map[string]*pb.Session
	GcWhiteListMu *sync.Mutex
	KubeLabs      map[string]*KubeLab
	KubeLabsMu    *sync.Mutex
	HealthChecker LessonHealthChecker

	// Allows us to disable GC for testing. Production code should leave this at
	// false
	DisableGC bool

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
func (ls *LessonScheduler) Start() error {
	// Ensure cluster is cleansed before we start the scheduler
	// TODO(mierdin): need to clearly document this behavior and warn to not edit kubernetes resources with the syringeManaged label
	ls.nukeFromOrbit()
	// I have taken this out now that garbage collection is in place. We should probably not have this in here, in case syringe panics, and then restarts, nuking everything.

	// Ensure our network CRD is in place (should fail silently if already exists)
	ls.createNetworkCrd()

	// TODO(mierdin): Maybe not an issue right now, but should consider if we should check if another Syringe is operating with
	// our configured ID.

	// Garbage collection
	if !ls.DisableGC {
		go func() {
			for {

				cleaned, err := ls.PurgeOldLessons()
				if err != nil {
					log.Error("Problem with GCing lessons")
				}

				for i := range cleaned {

					// Clean up local kubelab state
					ls.deleteKubelab(cleaned[i])

					// Send result to API server to clean up livelesson state
					ls.Results <- &LessonScheduleResult{
						Success:   true,
						Lesson:    nil,
						Uuid:      cleaned[i],
						Operation: OperationType_DELETE,
					}
				}
				time.Sleep(1 * time.Minute)

			}
		}()
	}

	// Handle incoming requests asynchronously
	var handlers = map[OperationType]interface{}{
		OperationType_CREATE: ls.handleRequestCREATE,
		OperationType_DELETE: ls.handleRequestDELETE,
		OperationType_MODIFY: ls.handleRequestMODIFY,
		OperationType_BOOP:   ls.handleRequestBOOP,
		OperationType_VERIFY: ls.handleRequestVERIFY,
	}
	for {
		newRequest := <-ls.Requests

		log.WithFields(log.Fields{
			"Operation": newRequest.Operation,
			"Uuid":      newRequest.Uuid,
			"Stage":     newRequest.Stage,
		}).Debug("Scheduler received new request. Sending to handle function.")

		go func() {
			handlers[newRequest.Operation].(func(*LessonScheduleRequest))(newRequest)
		}()
	}
	return nil
}

func (ls *LessonScheduler) setKubelab(uuid string, kl *KubeLab) {
	ls.KubeLabsMu.Lock()
	defer ls.KubeLabsMu.Unlock()
	ls.KubeLabs[uuid] = kl
}

func (ls *LessonScheduler) deleteKubelab(uuid string) {
	if _, ok := ls.KubeLabs[uuid]; !ok {
		return
	}
	ls.KubeLabsMu.Lock()
	defer ls.KubeLabsMu.Unlock()
	delete(ls.KubeLabs, uuid)
}

func (ls *LessonScheduler) configureStuff(nsName string, liveLesson *pb.LiveLesson, newRequest *LessonScheduleRequest) error {

	ls.killAllJobs(nsName, "config")

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

		job, err := ls.configureEndpoint(liveLesson.LiveEndpoints[i], newRequest)
		if err != nil {
			log.Errorf("Problem configuring endpoint %s", liveLesson.LiveEndpoints[i].Name)
			continue // TODO(mierdin): should quit entirely and return an error result to the channel
			// though this error is only immediate errors creating the job. This will succeed even if
			// the eventually configuration fails. See below for a better handle on configuration failures.
		}
		go func() {
			defer wg.Done()

			for i := 0; i < 120; i++ {
				completed, err := ls.isCompleted(job, newRequest)
				if err != nil {
					allGood = false
					return
				}

				time.Sleep(5 * time.Second)
				if completed {
					return
				}
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
func (ls *LessonScheduler) getVolumesConfiguration(lesson *pb.Lesson) ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initContainers := []corev1.Container{}

	lessonDir := strings.TrimPrefix(lesson.LessonDir, fmt.Sprintf("%s/", ls.SyringeConfig.CurriculumDir))

	if ls.SyringeConfig.CurriculumLocal {

		// Init container will mount the host directory as read-only, and copy entire contents into an emptyDir volume
		initContainers = append(initContainers, corev1.Container{
			Name:  "copy-local-files",
			Image: "bash",
			Command: []string{
				"bash",
			},
			Args: []string{
				"-c",
				fmt.Sprintf("cp -r %s-ro/lessons/ %s && adduser -D antidote && chown -R antidote:antidote %s",
					ls.SyringeConfig.CurriculumDir,
					ls.SyringeConfig.CurriculumDir,
					ls.SyringeConfig.CurriculumDir),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "host-volume",
					ReadOnly:  true,
					MountPath: fmt.Sprintf("%s-ro", ls.SyringeConfig.CurriculumDir),
				},
				{
					Name:      "local-copy",
					ReadOnly:  false,
					MountPath: ls.SyringeConfig.CurriculumDir,
				},
			},
		})

		// Add outer host volume, should be mounted read-only
		volumes = append(volumes, corev1.Volume{
			Name: "host-volume",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: ls.SyringeConfig.CurriculumDir,
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
			MountPath: ls.SyringeConfig.CurriculumDir,
			SubPath:   lessonDir,
		})

	} else {
		volumes = append(volumes, corev1.Volume{
			Name: "git-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "git-volume",
			ReadOnly:  false,
			MountPath: ls.SyringeConfig.CurriculumDir,
			SubPath:   lessonDir,
		})

		initContainers = append(initContainers, corev1.Container{
			Name:  "git-clone",
			Image: fmt.Sprintf("antidotelabs/githelper:%s", ls.BuildInfo["imageVersion"]),
			Args: []string{
				ls.SyringeConfig.CurriculumRepoRemote,
				ls.SyringeConfig.CurriculumRepoBranch,
				ls.SyringeConfig.CurriculumDir,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "git-volume",
					ReadOnly:  false,
					MountPath: ls.SyringeConfig.CurriculumDir,
				},
			},
		})
	}

	return volumes, volumeMounts, initContainers

}

func (ls *LessonScheduler) testEndpointReachability(ll *pb.LiveLesson) map[string]bool {

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

		// If no presentations, we can just test the first port in the additionalPorts list.
		if len(ep.Presentations) == 0 && len(ep.AdditionalPorts) != 0 {

			go func() {
				defer wg.Done()
				testResult := false

				log.Debugf("Performing basic connectivity test against endpoint %s via %s:%d", ep.Name, ep.Host, ep.AdditionalPorts[0])
				testResult = ls.HealthChecker.tcpTest(ep.Host, int(ep.AdditionalPorts[0]))

				if testResult {
					log.Debugf("%s is live at %s:%d", ep.Name, ep.Host, ep.AdditionalPorts[0])
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
				testResult = ls.HealthChecker.tcpTest(ep.Host, int(lp.Port))
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

// LessonHealthChecker describes a struct which offers a variety of reachability
// tests for lesson endpoints.
type LessonHealthChecker interface {
	sshTest(string, int) bool
	tcpTest(string, int) bool
}

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

// usesJupyterLabGuide is a helper function that lets us know if a lesson def uses a
// jupyter notebook as a lab guide in any stage.
func usesJupyterLabGuide(ld *pb.Lesson) bool {
	for i := range ld.Stages {
		if ld.Stages[i].JupyterLabGuide {
			return true
		}
	}

	return false
}
