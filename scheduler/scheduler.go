// Responsible for creating all resources for a lab. Pods, services, networks, etc.
package scheduler

import (
	"errors"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/def"
	crd "github.com/nre-learning/syringe/pkg/apis/kubernetes.com/v1"
	corev1 "k8s.io/api/core/v1"
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
					KubeLab:   nil,
					Uuid:      "",
					Operation: newRequest.Operation,
				}
			}
			ls.Results <- &LabScheduleResult{
				Success:   true,
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
					KubeLab:   nil,
					Uuid:      "",
					Operation: newRequest.Operation,
				}
			}
			ls.Results <- &LabScheduleResult{
				Success:   true,
				KubeLab:   nil,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
			}
		}

		// log.Infof("Created lab:\n%+v\n", newKubeLab)
		// log.Infof(newKubeLab.LabConnections["csrx1"])
		// lab1.TearDown()

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
		Networks:       map[string]*crd.Network{},
		Pods:           map[string]*corev1.Pod{},
		Services:       map[string]*corev1.Service{},
		LabConnections: map[string]string{},
	}

	// Create management network for "misc" stuff like notebooks
	_, err = ls.createNetwork("mgmt-net", req)
	if err != nil {
		return nil, err
	}

	// Create networks from connections property
	for c := range req.LabDef.Connections {
		connection := req.LabDef.Connections[c]
		newNet, err := ls.createNetwork(fmt.Sprintf("%s-%s-net", connection.A, connection.B), req)
		if err != nil {
			log.Error(err)
		}

		// log.Infof("About to add %v at index %s", &newNet, &newNet.ObjectMeta.Name)

		kl.Networks[newNet.ObjectMeta.Name] = newNet
	}

	// type LabDefinition struct {
	// 	LabName     string        `json:"labName" yaml:"labName"`
	// 	LabID       int32         `json:"labID" yaml:"labID"`
	// 	Devices     []*Device     `json:"devices" yaml:"devices"`
	// 	Connections []*Connection `json:"connections" yaml:"connections"`
	// }

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

	// log.Info("Creating service: csrx1")
	// service1, _ := ls.createService("csrx1svc", "1", "1")

	// service1Port, _ := getSSHServicePort(service1)

	// return &KubeLab{
	// 	Networks: map[string]*crd.Network{
	// 		network1.ObjectMeta.Name: &network1
	// 	},
	// 	Pods: []string{
	// 		pod1.ObjectMeta.Name,
	// 	},
	// 	Services: []string{
	// 		service1.ObjectMeta.Name,
	// 	},
	// 	LabConnections: map[string]string{
	// 		"csrx1": service1Port,
	// 	},
	// }, nil
	return kl, nil
}

// func (ls *LabScheduler) TearDown(l *KubeLab) error {

// 	// TODO(mierdin): Make sure the relevant maps here and in the API are updated when this happens.

// 	for n := range l.Services {
// 		ls.deleteService(l.Services[n])
// 	}

// 	for n := range l.Pods {
// 		ls.deletePod(l.Pods[n])
// 	}

// 	for n := range l.Networks {
// 		ls.deleteNetwork(l.Networks[n])
// 	}

// 	// TODO(mierdin): Delete namespace

// 	return nil

// }

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
	Networks       map[string]*crd.Network
	Pods           map[string]*corev1.Pod
	Services       map[string]*corev1.Service
	LabConnections map[string]string
}

// ToLiveLab exports a KubeLab as a generic LiveLab so the API can use it
func (kl *KubeLab) ToLiveLab() *pb.LiveLab {

	ret := pb.LiveLab{
		LabUUID:   kl.CreateRequest.Uuid,
		LabId:     kl.CreateRequest.LabDef.LabID,
		Endpoints: []*pb.LabEndpoint{}, //TODO(mierdin): obviously need to populate this
		Ready:     true,                //TODO(mierdin): will need to get a health check before validating this, or validate async
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

	// message LiveLab {
	// 	string LabUUID = 1;
	// 	int32 labId = 2;
	// 	repeated LabEndpoint endpoints  = 3;
	// 	bool ready = 4;
	//   }

	// type KubeLab struct {
	// 	Networks       []string
	// 	Pods           []string
	// 	Services       []string
	// 	LabConnections map[string]string
	// }

	// message LabEndpoint {
	// 	string name  = 1;

	// 	enum EndpointType {
	// 	  DEVICE = 0;        // A network device. Expected to be reachable via SSH or API on the listed port
	// 	  NOTEBOOK = 1;      // Jupyter server
	// 	  BLACKBOX = 2;      // Some kind of entity that the user doesn't have access to (i.e. for troubleshooting)
	// 	  LINUX = 3;         // Linux container we want to provide access to for tools
	// 	  CONFIGURATOR = 4;  // Configurator container that's responsible for bootstrapping the lab devices
	// 	}
	// 	EndpointType type = 2;
	// 	int32 port  = 3;
	// 	int32 api_port  = 4;
	//   }

	return &ret
}
