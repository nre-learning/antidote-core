// Responsible for creating all resources for a lab. Pods, services, networks, etc.
package scheduler

import (
	"errors"
	"fmt"
	"net"
	"strconv"
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
	LessonDefs    map[int32]*pb.LessonDef
	SyringeConfig *config.SyringeConfig
	GcWhiteList   map[string]*pb.Session
	GcWhiteListMu *sync.Mutex
	KubeLabs      map[string]*KubeLab
	KubeLabsMu    *sync.Mutex
	HealthChecker LessonHealthCheck

	// Allows us to disable GC for testing. Production code should leave this at
	// false
	DisableGC bool

	// Client for interacting with normal Kubernetes resources
	Client kubernetes.Interface

	// Client for creating CRD defintions
	ClientExt kubernetesExt.Interface

	// Client for creating instances of our network CRD
	ClientCrd kubernetesCrd.Interface
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
						LessonDef: nil,
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

	// Perform configuration changes for devices only
	var deviceEndpoints []*pb.LiveEndpoint
	for i := range liveLesson.LiveEndpoints {
		ep := liveLesson.LiveEndpoints[i]
		if ep.Type == pb.LiveEndpoint_DEVICE {
			deviceEndpoints = append(deviceEndpoints, ep)
		}
	}
	wg := new(sync.WaitGroup)
	wg.Add(len(deviceEndpoints))
	allGood := true
	for i := range deviceEndpoints {
		job, err := ls.configureDevice(deviceEndpoints[i], newRequest)
		if err != nil {
			log.Errorf("Problem configuring device %s", deviceEndpoints[i].Name)
			continue // TODO(mierdin): should quit entirely and return an error result to the channel
		}
		go func() {
			defer wg.Done()

			for i := 0; i < 120; i++ {
				completed, _ := ls.isCompleted(job, newRequest)
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
func (ls *LessonScheduler) getVolumesConfiguration() ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initContainers := []corev1.Container{}

	if ls.SyringeConfig.LessonsLocal {

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
					ls.SyringeConfig.LessonRepoDir,
					ls.SyringeConfig.LessonRepoDir,
					ls.SyringeConfig.LessonRepoDir),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "host-volume",
					ReadOnly:  true,
					MountPath: fmt.Sprintf("%s-ro", ls.SyringeConfig.LessonRepoDir),
				},
				{
					Name:      "local-copy",
					ReadOnly:  false,
					MountPath: ls.SyringeConfig.LessonRepoDir,
				},
			},
		})

		// Add outer host volume, should be mounted read-only
		volumes = append(volumes, corev1.Volume{
			Name: "host-volume",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: ls.SyringeConfig.LessonRepoDir,
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
			MountPath: ls.SyringeConfig.LessonRepoDir,
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
			MountPath: ls.SyringeConfig.LessonRepoDir,
		})

		initContainers = append(initContainers, corev1.Container{
			Name:  "git-clone",
			Image: "antidotelabs/githelper",
			Args: []string{
				ls.SyringeConfig.LessonRepoRemote,
				ls.SyringeConfig.LessonRepoBranch,
				ls.SyringeConfig.LessonRepoDir,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "git-volume",
					ReadOnly:  false,
					MountPath: ls.SyringeConfig.LessonRepoDir,
				},
			},
		})
	}

	return volumes, volumeMounts, initContainers

}

func (ls *LessonScheduler) testEndpointReachability(ll *pb.LiveLesson) map[string]bool {

	reachableMap := map[string]bool{}

	wg := new(sync.WaitGroup)
	wg.Add(len(ll.LiveEndpoints))

	var mapMutex = &sync.Mutex{}

	for d := range ll.LiveEndpoints {

		ep := ll.LiveEndpoints[d]

		go func() {
			defer wg.Done()

			testResult := false

			if ep.GetType() == pb.LiveEndpoint_DEVICE || ep.GetType() == pb.LiveEndpoint_UTILITY {
				log.Debugf("Performing SSH connectivity test against endpoint %s via %s:%d", ep.Name, ep.Host, ep.Port)
				testResult = ls.HealthChecker.sshTest(ep)
			} else if ep.GetType() == pb.LiveEndpoint_BLACKBOX {
				log.Debugf("Performing basic connectivity test against endpoint %s via %s:%d", ep.Name, ep.Host, ep.Port)
				testResult = ls.HealthChecker.tcpTest(ep)
			} else {
				testResult = true
			}
			mapMutex.Lock()
			defer mapMutex.Unlock()
			reachableMap[ep.Name] = testResult

		}()
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
	sshTest(*pb.LiveEndpoint) bool
	tcpTest(*pb.LiveEndpoint) bool
}

type LessonHealthCheck struct{}

func (lhc *LessonHealthCheck) sshTest(ep *pb.LiveEndpoint) bool {
	port := strconv.Itoa(int(ep.Port))
	sshConfig := &ssh.ClientConfig{
		User:            "antidote",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password("antidotepassword"),
		},
		Timeout: time.Second * 2,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", ep.Host, port), sshConfig)
	if err != nil {
		return false
	}
	defer conn.Close()

	log.Debugf("%s is live at %s:%s", ep.Name, ep.Host, port)
	return true
}

func (lhc *LessonHealthCheck) tcpTest(ep *pb.LiveEndpoint) bool {
	intPort := strconv.Itoa(int(ep.Port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", ep.Host, intPort), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	log.Debugf("done connect testing %s", ep.Host)
	return true
}

func HasDevices(ld *pb.LessonDef) bool {
	return len(ld.Devices) > 0
}

// usesJupyterLabGuide is a helper function that lets us know if a lesson def uses a
// jupyter notebook as a lab guide in any stage.
func usesJupyterLabGuide(ld *pb.LessonDef) bool {
	for i := range ld.Stages {
		if ld.Stages[i].JupyterLabGuide {
			return true
		}
	}

	return false
}

func getConnectivityInfo(svc *corev1.Service) (string, int, error) {

	var host string
	if svc.ObjectMeta.Labels["endpointType"] == "IFRAME" {
		if len(svc.Spec.ExternalIPs) > 0 {
			host = "svc.Spec.ExternalIPs[0]"
		} else {
			host = svc.Spec.ClusterIP
		}
		return host, int(svc.Spec.Ports[0].Port), nil
	} else {
		host = svc.Spec.ClusterIP
	}

	// We are only using the first port for the health check.
	if len(svc.Spec.Ports) == 0 {
		return "", 0, errors.New("unable to find port for service")
	} else {
		return host, int(svc.Spec.Ports[0].Port), nil
	}

}
