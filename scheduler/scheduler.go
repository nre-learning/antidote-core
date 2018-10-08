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
	"github.com/golang/protobuf/ptypes/timestamp"
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
	OperationType_BOOP   OperationType = 3
	OperationType_GC     OperationType = 4
	// _typePortMap                       = map[string]int32{
	// 	"DEVICE":  22,
	// 	"UTILITY": 22,
	// 	"IFRAME":  8888,
	// }
	defaultGitFileMode int32 = 0755
	kubeLabs                 = map[string]*KubeLab{}
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
	GCLessons []string
}

type LessonScheduler struct {
	KubeConfig *rest.Config
	Requests   chan *LessonScheduleRequest
	Results    chan *LessonScheduleResult
	LessonDefs map[int32]*def.LessonDefinition
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
	go func() {
		for {

			cleaned, err := ls.purgeOldLessons()
			if err != nil {
				log.Error("Problem with GCing lessons")
			}

			if len(cleaned) > 0 {
				ls.Results <- &LessonScheduleResult{
					Success:   true,
					LessonDef: nil,
					KubeLab:   nil,
					Uuid:      "",
					Operation: OperationType_GC,
					GCLessons: cleaned,
				}
			}

			time.Sleep(1 * time.Minute)

		}
	}()

	// Handle incoming requests asynchronously
	for {
		newRequest := <-ls.Requests
		go ls.handleRequest(newRequest)
		log.Debugf("Scheduler received new request - %v", newRequest)
	}

	return nil
}

func (ls *LessonScheduler) handleRequest(newRequest *LessonScheduleRequest) {
	nsName := fmt.Sprintf("%d-%s-ns", newRequest.LessonDef.LessonID, newRequest.Session)
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

		tries := 0
		for {
			time.Sleep(1 * time.Second)

			if tries > 600 {
				log.Errorf("Timeout waiting for lesson %d to become reachable", newRequest.LessonDef.LessonID)
				ls.Results <- &LessonScheduleResult{
					Success:   false,
					LessonDef: newRequest.LessonDef,
					KubeLab:   newKubeLab,
					Uuid:      newRequest.Uuid,
					Operation: newRequest.Operation,
					Stage:     newRequest.Stage,
				}
				return
			}

			if !isReachable(liveLesson) {
				tries++
				continue
			}
			break
		}

		if newRequest.LessonDef.TopologyType == "custom" {
			log.Infof("Performing configuration for new instance of lesson %d", newRequest.LessonDef.LessonID)
			ls.configureStuff(nsName, liveLesson, newRequest)
		} else {
			log.Infof("Skipping configuration of new instance of lesson %d", newRequest.LessonDef.LessonID)
		}

		kubeLabs[newRequest.Uuid] = newKubeLab

		ls.Results <- &LessonScheduleResult{
			Success:   true,
			LessonDef: newRequest.LessonDef,
			KubeLab:   newKubeLab,
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
			Stage:     newRequest.Stage,
		}
	} else if newRequest.Operation == OperationType_DELETE {
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

		kubeLabs[newRequest.Uuid].CreateRequest = newRequest
		liveLesson := kubeLabs[newRequest.Uuid].ToLiveLesson()

		if newRequest.LessonDef.TopologyType == "custom" {
			log.Infof("Performing configuration of modified instance of lesson %d", newRequest.LessonDef.LessonID)
			ls.configureStuff(nsName, liveLesson, newRequest)
		} else {
			log.Infof("Skipping configuration of modified instance of lesson %d", newRequest.LessonDef.LessonID)
		}

		nsName := fmt.Sprintf("%d-%s-ns", newRequest.LessonDef.LessonID, newRequest.Session)

		err := ls.boopNamespace(nsName)
		if err != nil {
			log.Errorf("Problem modify-booping %s: %v", nsName, err)
		}

		ls.Results <- &LessonScheduleResult{
			Success:   true,
			LessonDef: newRequest.LessonDef,
			KubeLab:   kubeLabs[newRequest.Uuid],
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
			Stage:     newRequest.Stage,
		}

	} else if newRequest.Operation == OperationType_BOOP {
		nsName := fmt.Sprintf("%d-%s-ns", newRequest.LessonDef.LessonID, newRequest.Session)

		err := ls.boopNamespace(nsName)
		if err != nil {
			log.Errorf("Problem boop-booping %s: %v", nsName, err)
		}
	}

	log.Debug("Result sent. Now waiting for next schedule request...")
}

func (ls *LessonScheduler) configureStuff(nsName string, liveLesson *pb.LiveLesson, newRequest *LessonScheduleRequest) error {
	ls.killAllJobs(nsName)

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
				completed, _ := ls.isCompleted(job, newRequest)
				time.Sleep(1 * time.Second)
				if completed {
					return
				}
			}
		}()

	}

	wg.Wait()

	// for {
	// 	time.Sleep(1 * time.Second)

	// 	if tries > 600 {
	// 		log.Errorf("Timeout waiting for lesson %d to become reachable", newRequest.LessonDef.LessonID)
	// 		ls.Results <- &LessonScheduleResult{
	// 			Success:   false,
	// 			LessonDef: newRequest.LessonDef,
	// 			KubeLab:   newKubeLab,
	// 			Uuid:      newRequest.Uuid,
	// 			Operation: newRequest.Operation,
	// 			Stage:     newRequest.Stage,
	// 		}
	// 		return
	// 	}

	// 	if !isReachable(liveLesson) {
	// 		tries++
	// 		continue
	// 	}
	// 	break
	// }

	return nil
}

func (ls *LessonScheduler) createKubeLab(req *LessonScheduleRequest) (*KubeLab, error) {

	ns, err := ls.createNamespace(req)
	if err != nil {
		log.Error(err)
	}

	// Lock it doooooooown
	ls.createNetworkPolicy(ns.ObjectMeta.Name)

	kl := &KubeLab{
		Namespace:      ns,
		CreateRequest:  req,
		Networks:       map[string]*crd.NetworkAttachmentDefinition{},
		Pods:           map[string]*corev1.Pod{},
		Services:       map[string]*corev1.Service{},
		LabConnections: map[string]string{},
	}

	// TODO(mierdin): is this still needed?
	// _, err = ls.createNetwork("mgmt-net", req, false, "")
	// if err != nil {
	// 	log.Error(err)
	// }

	// Create our configmap for the initContainer for cloning the antidote repo
	ls.createGitConfigMap(ns.ObjectMeta.Name)

	// Only bother making connections and device pod/services if we have a custom topology
	log.Infof("New KubeLab for lesson %d is of TopologyType %s", kl.CreateRequest.LessonDef.LessonID, kl.CreateRequest.LessonDef.TopologyType)
	if kl.CreateRequest.LessonDef.TopologyType == "custom" {

		log.Debug("Creating devices and connections")

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
				device,
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

	} else {
		log.Debug("Not creating devices and connections")
	}

	// Create pods and services for utility containers
	for d := range req.LessonDef.Utilities {

		utility := req.LessonDef.Utilities[d]
		newPod, err := ls.createPod(
			utility,
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

	// Create pods and services for black box containers
	for d := range req.LessonDef.Blackboxes {

		blackbox := req.LessonDef.Blackboxes[d]
		newPod, err := ls.createPod(
			blackbox,
			pb.Endpoint_BLACKBOX,
			getMemberNetworks(blackbox.Name, req.LessonDef.Connections),
			req,
		)
		if err != nil {
			log.Error(err)
		}
		kl.Pods[newPod.ObjectMeta.Name] = newPod

		if len(newPod.Spec.Containers[0].Ports) > 0 {
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

	if req.LessonDef.Stages[req.Stage].IframeResource.URI != "" {

		iframePod, _ := ls.createPod(
			&def.Endpoint{
				Name:  req.LessonDef.Stages[req.Stage].IframeResource.Name,
				Image: "antidotelabs/jupyter",
				Ports: []int32{},
			},
			pb.Endpoint_IFRAME,
			[]string{},
			req,
		)
		kl.Pods[iframePod.ObjectMeta.Name] = iframePod

		// Create service for this pod
		iframeSvc, _ := ls.createService(
			iframePod,
			req,
		)
		kl.Services[iframeSvc.ObjectMeta.Name] = iframeSvc
	}

	return kl, nil
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

	ts, _ := strconv.ParseInt(kl.Namespace.ObjectMeta.Labels["lastAccessed"], 10, 64)

	ret := pb.LiveLesson{
		LessonUUID:    kl.CreateRequest.Uuid,
		LessonId:      kl.CreateRequest.LessonDef.LessonID,
		Endpoints:     []*pb.Endpoint{},
		LessonStage:   kl.CreateRequest.Stage,
		LessonDiagram: kl.CreateRequest.LessonDef.LessonDiagram,
		LessonVideo:   kl.CreateRequest.LessonDef.LessonVideo,
		LabGuide:      kl.CreateRequest.LessonDef.Stages[kl.CreateRequest.Stage].LabGuide,
		CreatedTime: &timestamp.Timestamp{
			Seconds: ts,
		},

		// Previously we were overriding this value, so it was set to false, and then the API would perform the health check.
		// Now, the health check is done by the scheduler, and is only returned to the API when everything is ready. So we
		// need this to be set to true.
		//
		// You may consider moving this field to kubelab or something.
		Ready: true,
	}

	if kl.CreateRequest.LessonDef.TopologyType == "shared" {
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
			if kl.Pods[p].ObjectMeta.Name == kl.Services[s].ObjectMeta.Name {
				podBuddy = kl.Pods[p]
				break
			}
		}

		// TODO(mierdin): handle if podbuddy is still empty

		host, port, _ := getConnectivityInfo(kl.Services[s])
		// portInt, _ := strconv.Atoi(port)

		endpoint := &pb.Endpoint{
			Name:        podBuddy.ObjectMeta.Name,
			Type:        pb.Endpoint_EndpointType(pb.Endpoint_EndpointType_value[podBuddy.Labels["endpointType"]]),
			Host:        host,
			Port:        int32(port),
			Sshuser:     kl.Services[s].ObjectMeta.Labels["sshUser"],
			Sshpassword: kl.Services[s].ObjectMeta.Labels["sshPassword"],
			// ApiPort
		}

		if endpoint.Type == pb.Endpoint_IFRAME {
			endpoint.IframeDetails = &pb.IFDetails{
				Name:     kl.CreateRequest.LessonDef.Stages[kl.CreateRequest.Stage].IframeResource.Name,
				URI:      kl.CreateRequest.LessonDef.Stages[kl.CreateRequest.Stage].IframeResource.URI,
				Protocol: kl.CreateRequest.LessonDef.Stages[kl.CreateRequest.Stage].IframeResource.Protocol,
				Port:     kl.Services[s].Spec.Ports[0].NodePort,
			}
		}

		ret.Endpoints = append(ret.Endpoints, endpoint)
	}

	ret.LabGuide = kl.CreateRequest.LessonDef.Stages[kl.CreateRequest.Stage].LabGuide

	return &ret
}

func (ls *LessonScheduler) createGitConfigMap(nsName string) error {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
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

	reachableMap := map[string]bool{}

	wg := new(sync.WaitGroup)
	wg.Add(len(ll.Endpoints))

	var mapMutex = &sync.Mutex{}

	for d := range ll.Endpoints {

		ep := ll.Endpoints[d]

		go func() {
			defer wg.Done()
			log.Debugf("Connectivity testing endpoint %s via %s:%d", ep.Name, ep.Host, ep.Port)

			testResult := false

			if ep.GetType() == pb.Endpoint_DEVICE || ep.GetType() == pb.Endpoint_UTILITY {
				testResult = sshTest(ep)
			} else if ep.GetType() == pb.Endpoint_IFRAME || ep.GetType() == pb.Endpoint_BLACKBOX {
				testResult = connectTest(ep)
			}
			mapMutex.Lock()
			defer mapMutex.Unlock()
			reachableMap[ep.Name] = testResult

		}()
	}
	wg.Wait()

	log.Debugf("Livelesson %s health check results: %v", ll.LessonUUID, reachableMap)

	for _, reachable := range reachableMap {
		if !reachable {
			return false
		}
	}

	return true
}

func sshTest(ep *pb.Endpoint) bool {
	intPort := strconv.Itoa(int(ep.Port))
	sshConfig := &ssh.ClientConfig{
		User:            ep.Sshuser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(ep.Sshpassword),
		},
		Timeout: time.Second * 2,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", ep.Host, intPort), sshConfig)
	if err != nil {
		return false
	}
	defer conn.Close()

	log.Debugf("done ssh testing %s", ep.Host)
	return true
}

func connectTest(ep *pb.Endpoint) bool {
	intPort := strconv.Itoa(int(ep.Port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", ep.Host, intPort), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	log.Debugf("done connect testing %s", ep.Host)
	return true
}
