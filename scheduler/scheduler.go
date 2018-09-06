// Responsible for creating all resources for a lab. Pods, services, networks, etc.
package scheduler

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/def"
	crd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type OperationType int32

var (
	OperationType_CREATE OperationType = 0
	OperationType_DELETE OperationType = 1
	OperationType_MODIFY OperationType = 2
	typePortMap                        = map[string]int32{
		"DEVICE":   22,
		"UTILITY":  22,
		"NOTEBOOK": 8888,
	}
	defaultGitFileMode int32 = 0755
)

type LessonScheduleRequest struct {
	LessonDef *def.LessonDefinition
	Operation OperationType
	Uuid      string
	Session   string
	Stage     int32
}

type LessonScheduleResult struct {
	Success   bool
	Stage     int32
	LessonDef *def.LessonDefinition
	Operation OperationType
	Message   string
	KubeLab   *KubeLab
	Uuid      string
	Session   string
}

type LessonScheduler struct {
	Config     *rest.Config
	Requests   chan *LessonScheduleRequest
	Results    chan *LessonScheduleResult
	LessonDefs map[int32]*def.LessonDefinition
}

// Start is meant to be run as a goroutine. The "requests" channel will wait for new requests, attempt to schedule them,
// and put a results message on the "results" channel when finished (success or fail)
func (ls *LessonScheduler) Start() error {

	// Ensure cluster is cleansed before we start the scheduler
	// TODO(mierdin): need to clearly document this behavior and warn to not edit kubernetes resources with the syringeManaged label
	// ls.nukeFromOrbit()

	// Ensure our network CRD is in place (should fail silently if already exists)
	ls.createNetworkCrd()

	// Need to start a garbage collector that watches for livelessons to reach a certain age and delete them (and clear from memory here).
	// Might need to look into some kind of activity check so you don't kill something that was actually in use
	// You should also expose the delete functionality so the javascript can send a quit signal for a lesson but you'll want to make sure
	// people can't kill others lessons.

	for {
		newRequest := <-ls.Requests

		log.Debug("Scheduler received new request")
		log.Debug(newRequest)

		if newRequest.Operation == OperationType_CREATE {
			newKubeLab, err := ls.createKubeLab(newRequest)
			if err != nil {
				log.Errorf("Error creating lesson: %s", err)
				ls.Results <- &LessonScheduleResult{
					Success:   false,
					LessonDef: newRequest.LessonDef,
					KubeLab:   nil,
					Uuid:      "",
					Operation: newRequest.Operation,
				}
			}

			liveLesson := newKubeLab.ToLiveLesson()

			// TODO(mierdin) need to add timeout
			for {
				time.Sleep(1 * time.Second)

				// TODO(mierdin): tcp connection test isn't enough. Need to do SSH and ensure we can at least get that
				if !isReachable(liveLesson) {
					continue
				}
				break
			}

			// Perform configuration changes for devices only
			var deviceEndpoints []*pb.Endpoint
			for i := range liveLesson.Endpoints {
				ep := liveLesson.Endpoints[i]
				if ep.Type == pb.Endpoint_DEVICE {
					deviceEndpoints = append(deviceEndpoints, ep)
				}
			}
			wg := new(sync.WaitGroup)
			wg.Add(len(deviceEndpoints))
			for i := range deviceEndpoints {
				job, err := ls.configureDevice(deviceEndpoints[i], newRequest)
				if err != nil {
					log.Errorf("Problem configuring device %s", deviceEndpoints[i].Name)
					continue // TODO(mierdin): should quit entirely and return an error result to the channel
				}
				go func() {
					defer wg.Done()

					// TODO(mierdin): Add timeout
					for {
						time.Sleep(2 * time.Second)
						completed, _ := ls.isCompleted(job, newRequest)
						if completed {
							break
						}
					}
				}()

			}

			wg.Wait()

			ls.Results <- &LessonScheduleResult{
				Success:   true,
				LessonDef: newRequest.LessonDef,
				KubeLab:   newKubeLab,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
			}
		} else if newRequest.Operation == OperationType_DELETE {
			nsName := fmt.Sprintf("%d-%s-ns", newRequest.LessonDef.LessonID, newRequest.Session)
			err := ls.deleteNamespace(nsName)
			if err != nil {
				log.Errorf("Error deleting lesson: %s", err)
				ls.Results <- &LessonScheduleResult{
					Success:   false,
					LessonDef: newRequest.LessonDef,
					KubeLab:   nil,
					Uuid:      "",
					Operation: newRequest.Operation,
				}
			}
			ls.Results <- &LessonScheduleResult{
				Success:   true,
				LessonDef: newRequest.LessonDef,
				KubeLab:   nil,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
			}
		} else if newRequest.Operation == OperationType_MODIFY {
			//
		}

		log.Debug("Result sent. Now waiting for next schedule request...")
	}

	return nil
}

func (ls *LessonScheduler) createKubeLab(req *LessonScheduleRequest) (*KubeLab, error) {

	ns, err := ls.createNamespace(req)
	if err != nil {
		log.Error(err)
	}

	kl := &KubeLab{
		Namespace:      ns,
		CreateRequest:  req,
		Networks:       map[string]*crd.NetworkAttachmentDefinition{},
		Pods:           map[string]*corev1.Pod{},
		Services:       map[string]*corev1.Service{},
		LabConnections: map[string]string{},
	}

	// Create management network for "misc" stuff like notebooks EDIT not needed anymore
	_, err = ls.createNetwork("mgmt-net", req, false, "")
	if err != nil {
		log.Error(err)
	}

	// Create our configmap for the initContainer for cloning the antidote repo
	ls.createGitConfigMap(ns.ObjectMeta.Name)

	// Only bother making connections and device pod/services if we're not using the
	// shared topology
	if !kl.CreateRequest.LessonDef.SharedTopology {

		// Create networks from connections property
		for c := range req.LessonDef.Connections {
			connection := req.LessonDef.Connections[c]
			newNet, err := ls.createNetwork(fmt.Sprintf("%s-%s-net", connection.A, connection.B), req, true, connection.Subnet)
			if err != nil {
				log.Error(err)
			}

			// log.Infof("About to add %v at index %s", &newNet, &newNet.ObjectMeta.Name)

			kl.Networks[newNet.ObjectMeta.Name] = newNet
		}

		// Create pods and services for devices
		for d := range req.LessonDef.Devices {

			// Create pods from devices property
			device := req.LessonDef.Devices[d]
			newPod, err := ls.createPod(
				device.Name,
				device.Image,
				pb.Endpoint_DEVICE,
				getMemberNetworks(device.Name, req.LessonDef.Connections),
				req,
			)
			if err != nil {
				log.Error(err)
			}
			kl.Pods[newPod.ObjectMeta.Name] = newPod

			// Create service for this pod
			newSvc, err := ls.createService(
				newPod,
				req,
			)
			if err != nil {
				log.Error(err)
			}
			kl.Services[newSvc.ObjectMeta.Name] = newSvc
		}

		// Create pods and services for utility containers
		for d := range req.LessonDef.Utilities {

			utility := req.LessonDef.Utilities[d]
			newPod, err := ls.createPod(
				utility.Name,
				utility.Image,
				pb.Endpoint_UTILITY,
				getMemberNetworks(utility.Name, req.LessonDef.Connections),
				req,
			)
			if err != nil {
				log.Error(err)
			}
			kl.Pods[newPod.ObjectMeta.Name] = newPod

			// Create service for this pod
			newSvc, err := ls.createService(
				newPod,
				req,
			)
			if err != nil {
				log.Error(err)
			}
			kl.Services[newSvc.ObjectMeta.Name] = newSvc
		}

	}

	// TODO(mierdin): convert this field back to boolean and make sure it gets passed to the API. Then use the below:
	// The path is deterministic, so we don't need the user to specify the path
	if true {
		// if req.LessonDef.Stages[req.Stage].Notebook {

		notebookPod, _ := ls.createPod(
			"notebook",
			"antidotelabs/jupyter",
			pb.Endpoint_NOTEBOOK,
			[]string{"mgmt-net"},
			req,
		)
		kl.Pods[notebookPod.ObjectMeta.Name] = notebookPod

		// Create service for this pod
		notebookSvc, _ := ls.createService(
			notebookPod,
			req,
		)
		kl.Services[notebookSvc.ObjectMeta.Name] = notebookSvc
	}

	return kl, nil
}

func getSSHServicePort(svc *corev1.Service) (string, error) {
	for p := range svc.Spec.Ports {

		// TODO should set port name consistently via syringe, and look up via name instead here
		// so you can map to port and apiport in LiveLab entity
		if svc.Spec.Ports[p].Name == "primaryport" {

			// TODO should also detect an undefined NodePort, kind of like this
			// if svc.Spec.Ports[p].NodePort == nil {
			// 	log.Error("NodePort undefined for service")
			// 	return "", errors.New("unable to find NodePort for service")
			// }

			return strconv.Itoa(int(svc.Spec.Ports[p].NodePort)), nil
		}
	}
	log.Error("unable to find NodePort for service")
	return "", errors.New("unable to find NodePort for service")
}

// KubeLab is the collection of kubernetes resources that makes up a lab instance
type KubeLab struct {
	Namespace      *corev1.Namespace
	CreateRequest  *LessonScheduleRequest // The request that originally resulted in this KubeLab
	Networks       map[string]*crd.NetworkAttachmentDefinition
	Pods           map[string]*corev1.Pod
	Services       map[string]*corev1.Service
	LabConnections map[string]string
}

// ToLiveLesson exports a KubeLab as a generic LiveLesson so the API can use it
func (kl *KubeLab) ToLiveLesson() *pb.LiveLesson {

	ret := pb.LiveLesson{
		LessonUUID:  kl.CreateRequest.Uuid,
		LessonId:    kl.CreateRequest.LessonDef.LessonID,
		Endpoints:   []*pb.Endpoint{},
		LessonStage: kl.CreateRequest.Stage,

		// Previously we were overriding this value, so it was set to false, and then the API would perform the health check.
		// Now, the health check is done by the scheduler, and is only returned to the API when everything is ready. So we
		// need this to be set to true.
		//
		// You may consider moving this field to kubelab or something.
		Ready: true,
	}

	if kl.CreateRequest.LessonDef.SharedTopology {
		ret.Endpoints = []*pb.Endpoint{
			{
				Name: "vqfx1",
				Type: pb.Endpoint_DEVICE,
				Port: 30021,
			}, {
				Name: "vqfx2",
				Type: pb.Endpoint_DEVICE,
				Port: 30022,
			}, {
				Name: "vqfx3",
				Type: pb.Endpoint_DEVICE,
				Port: 30023,
			},
		}
	}

	for s := range kl.Services {

		// find corresponding pod for this service
		var podBuddy *corev1.Pod
		for p := range kl.Pods {
			if fmt.Sprintf("%s-svc", kl.Pods[p].ObjectMeta.Name) == kl.Services[s].ObjectMeta.Name {
				podBuddy = kl.Pods[p]
				break
			}
		}

		port, _ := getSSHServicePort(kl.Services[s])
		portInt, _ := strconv.Atoi(port)

		endpoint := &pb.Endpoint{
			Name: podBuddy.ObjectMeta.Name,
			Type: pb.Endpoint_EndpointType(pb.Endpoint_EndpointType_value[podBuddy.Labels["endpointType"]]),
			Port: int32(portInt),
			// ApiPort
		}
		ret.Endpoints = append(ret.Endpoints, endpoint)
	}

	ret.LabGuide = kl.CreateRequest.LessonDef.Stages[kl.CreateRequest.Stage].LabGuide

	return &ret
}

func (ls *LessonScheduler) createGitConfigMap(nsName string) error {

	coreclient, err := corev1client.NewForConfig(ls.Config)
	if err != nil {
		panic(err)
	}

	gitScript := `#!/bin/sh -e
REPO=$1
REF=$2
DIR=$3
# Init Containers will re-run on Pod restart. Remove the directory's contents
# and reprovision when this happens.
if [ -d "$DIR" ]; then
	rm -rf $( find $DIR -mindepth 1 )
fi
git clone $REPO $DIR
cd $DIR
git reset --hard $REF`

	svc := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-clone",
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
			},
		},
		Data: map[string]string{
			"git-clone.sh": gitScript,
		},
	}

	result, err := coreclient.ConfigMaps(nsName).Create(svc)
	if err == nil {
		log.Infof("Created configmap: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("ConfigMap %s already exists.", "git-clone")
		return nil
	} else {
		log.Errorf("Error creating configmap: %s", err)
		return err
	}

	return nil
}

func isReachable(ll *pb.LiveLesson) bool {
	for d := range ll.Endpoints {
		ep := ll.Endpoints[d]

		if ep.GetType() == pb.Endpoint_DEVICE {
			if sshTest(ep.Port, "VR-netlab9") {
				log.Debugf("%s health check passed on port %d", ep.Name, ep.Port)
			} else {
				log.Debugf("%s health check failed on port %d", ep.Name, ep.Port)
				return false
			}
		} else if ep.GetType() == pb.Endpoint_NOTEBOOK {
			if connectTest(ep.Port) {
				log.Debugf("%s health check passed on port %d", ep.Name, ep.Port)
			} else {
				log.Debugf("%s health check failed on port %d", ep.Name, ep.Port)
				return false
			}
		} else if ep.GetType() == pb.Endpoint_UTILITY {
			if sshTest(ep.Port, "antidotepassword") {
				log.Debugf("%s health check passed on port %d", ep.Name, ep.Port)
			} else {
				log.Debugf("%s health check failed on port %d", ep.Name, ep.Port)
				return false
			}
		}

	}
	return true
}

func sshTest(port int32, password string) bool {
	intPort := strconv.Itoa(int(port))

	sshConfig := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout: time.Second * 2,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("vip.labs.networkreliability.engineering:%s", intPort), sshConfig)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}

func connectTest(port int32) bool {
	intPort := strconv.Itoa(int(port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("vip.labs.networkreliability.engineering:%s", intPort), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
