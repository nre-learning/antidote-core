// Responsible for creating all resources for a lab. Pods, services, networks, etc.
package scheduler

import (
	"errors"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/def"
	crd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type OperationType int32

const (
	OperationType_CREATE OperationType = 0
	OperationType_DELETE OperationType = 1
)

type LabScheduleRequest struct {
	LabDef    *def.LabDefinition
	Operation OperationType
	Uuid      string
	Session   string
}

type LabScheduleResult struct {
	Success   bool
	LabDef    *def.LabDefinition
	Operation OperationType
	Message   string
	KubeLab   *KubeLab
	Uuid      string
	Session   string
}

type LabScheduler struct {
	Config   *rest.Config
	Requests chan *LabScheduleRequest
	Results  chan *LabScheduleResult
	LabDefs  map[int32]*def.LabDefinition
}

// Start is meant to be run as a goroutine. The "requests" channel will wait for new requests, attempt to schedule them,
// and put a results message on the "results" channel when finished (success or fail)
func (ls *LabScheduler) Start() error {

	// Ensure cluster is cleansed before we start the scheduler
	// TODO(mierdin): need to clearly document this behavior and warn to not edit kubernetes resources with the syringeManaged label
	ls.nukeFromOrbit()

	// Ensure our network CRD is in place (should fail silently if already exists)
	ls.createNetworkCrd()

	// Need to start a garbage collector that watches for livelabs to reach a certain age and delete them (and clear from memory here).
	// Might need to look into some kind of activity check so you don't kill something that was actually in use
	// You should also expose the delete functionality so the javascript can send a quit signal for a lab but you'll want to make sure
	// people can't kill others labs.

	for {
		newRequest := <-ls.Requests

		log.Debug("Scheduler received new request")
		log.Debug(newRequest)

		if newRequest.Operation == OperationType_CREATE {
			newKubeLab, err := ls.createKubeLab(newRequest)
			if err != nil {
				log.Errorf("Error creating lab: %s", err)
				ls.Results <- &LabScheduleResult{
					Success:   false,
					LabDef:    newRequest.LabDef,
					KubeLab:   nil,
					Uuid:      "",
					Operation: newRequest.Operation,
				}
			}
			ls.Results <- &LabScheduleResult{
				Success:   true,
				LabDef:    newRequest.LabDef,
				KubeLab:   newKubeLab,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
			}
		} else if newRequest.Operation == OperationType_DELETE {
			nsName := fmt.Sprintf("%d-%s-ns", newRequest.LabDef.LabID, newRequest.Session)
			err := ls.deleteNamespace(nsName)
			if err != nil {
				log.Errorf("Error creating lab: %s", err)
				ls.Results <- &LabScheduleResult{
					Success:   false,
					LabDef:    newRequest.LabDef,
					KubeLab:   nil,
					Uuid:      "",
					Operation: newRequest.Operation,
				}
			}
			ls.Results <- &LabScheduleResult{
				Success:   true,
				LabDef:    newRequest.LabDef,
				KubeLab:   nil,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
			}
		}

		log.Debug("Result sent. Now waiting for next schedule request...")
	}

	return nil
}

func (ls *LabScheduler) createKubeLab(req *LabScheduleRequest) (*KubeLab, error) {

	ns, err := ls.createNamespace(req)
	if err != nil {
		log.Error("failed to create namespace, not creating kubelab")
		return nil, err
	}

	kl := &KubeLab{
		Namespace:      ns,
		CreateRequest:  req,
		Networks:       map[string]*crd.NetworkAttachmentDefinition{},
		Pods:           map[string]*corev1.Pod{},
		Services:       map[string]*corev1.Service{},
		LabConnections: map[string]string{},
	}

	// Create management network for "misc" stuff like notebooks
	_, err = ls.createNetwork("mgmt-net", req, false, "")
	if err != nil {
		return nil, err
	}

	// Create our configmap for the initContainer for cloning the antidote repo
	ls.createGitConfigMap(ns.ObjectMeta.Name)

	// Only bother making connections and device pod/services if we're not using the
	// shared topology
	if !kl.CreateRequest.LabDef.SharedTopology {

		// Create networks from connections property
		for c := range req.LabDef.Connections {
			connection := req.LabDef.Connections[c]
			newNet, err := ls.createNetwork(fmt.Sprintf("%s-%s-net", connection.A, connection.B), req, true, connection.Subnet)
			if err != nil {
				log.Error(err)
			}

			// log.Infof("About to add %v at index %s", &newNet, &newNet.ObjectMeta.Name)

			kl.Networks[newNet.ObjectMeta.Name] = newNet
		}

		// Create pods and services for devices
		for d := range req.LabDef.Devices {

			// Create pods from devices property
			device := req.LabDef.Devices[d]
			newPod, _ := ls.createPod(
				device.Name,
				device.Image,
				pb.LabEndpoint_DEVICE,
				getMemberNetworks(device, req.LabDef.Connections),
				req,
			)
			kl.Pods[newPod.ObjectMeta.Name] = newPod

			// Create service for this pod
			newSvc, _ := ls.createService(
				newPod,
				req,
			)
			kl.Services[newSvc.ObjectMeta.Name] = newSvc
		}
	}
	if req.LabDef.Notebook {

		notebookPod, _ := ls.createPod(
			"notebook",
			"antidotelabs/jupyter",
			pb.LabEndpoint_NOTEBOOK,
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
	CreateRequest  *LabScheduleRequest // The request that originally resulted in this KubeLab
	Networks       map[string]*crd.NetworkAttachmentDefinition
	Pods           map[string]*corev1.Pod
	Services       map[string]*corev1.Service
	LabConnections map[string]string
}

// ToLiveLab exports a KubeLab as a generic LiveLab so the API can use it
func (kl *KubeLab) ToLiveLab() *pb.LiveLab {

	ret := pb.LiveLab{
		LabUUID:   kl.CreateRequest.Uuid,
		LabId:     kl.CreateRequest.LabDef.LabID,
		Endpoints: []*pb.LabEndpoint{},
		Ready:     false, // Set to false for now, will update elsewhere in a health check
	}

	if kl.CreateRequest.LabDef.SharedTopology {
		ret.Endpoints = []*pb.LabEndpoint{
			{
				Name: "vqfx1",
				Type: pb.LabEndpoint_DEVICE,
				Port: 30021,
			}, {
				Name: "vqfx2",
				Type: pb.LabEndpoint_DEVICE,
				Port: 30022,
			}, {
				Name: "vqfx3",
				Type: pb.LabEndpoint_DEVICE,
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

		endpoint := &pb.LabEndpoint{
			Name: podBuddy.ObjectMeta.Name,
			Type: pb.LabEndpoint_EndpointType(pb.LabEndpoint_EndpointType_value[podBuddy.Labels["endpointType"]]),
			Port: int32(portInt),
			// ApiPort
		}
		ret.Endpoints = append(ret.Endpoints, endpoint)
	}

	return &ret
}

func (ls *LabScheduler) createGitConfigMap(nsName string) error {

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
