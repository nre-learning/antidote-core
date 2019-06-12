package scheduler

import (
	"fmt"
	"strconv"

	"github.com/golang/protobuf/ptypes/timestamp"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	crd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
)

// KubeLab is the collection of kubernetes resources that makes up a lab instance
type KubeLab struct {
	Namespace          *corev1.Namespace
	CreateRequest      *LessonScheduleRequest // The request that originally resulted in this KubeLab
	Networks           map[string]*crd.NetworkAttachmentDefinition
	Pods               map[string]*corev1.Pod
	Services           map[string]*corev1.Service
	Ingresses          map[string]*v1beta1.Ingress
	Status             pb.Status
	ReachableEndpoints []string // endpoint names
	CurrentStage       int32
	HealthyTests       int
	TotalTests         int
}

// ToProtoKubeLab is a converter function that transforms a native KubeLab struct instance
// into a protobuf-based KubeLab instance. Not all fields can be modeled in protobuf, so this
// function mostly just uses the name of Kubernetes objects in lieu of the actual object.
func (kl *KubeLab) ToProtoKubeLab() *pb.KubeLab {

	networks := []string{}
	for k, _ := range kl.Networks {
		networks = append(networks, k)
	}

	pods := []string{}
	for k, _ := range kl.Pods {
		pods = append(pods, k)
	}

	services := []string{}
	for k, _ := range kl.Services {
		services = append(services, k)
	}

	ingresses := []string{}
	for k, _ := range kl.Ingresses {
		ingresses = append(ingresses, k)
	}

	ts := &timestamp.Timestamp{
		Seconds: kl.CreateRequest.Created.Unix(),
	}

	return &pb.KubeLab{
		Namespace: kl.Namespace.ObjectMeta.Name,
		CreateRequest: &pb.LessonScheduleRequest{
			Lesson:        kl.CreateRequest.Lesson,
			OperationType: int32(kl.CreateRequest.Operation),
			Uuid:          kl.CreateRequest.Uuid,
			Stage:         kl.CreateRequest.Stage,
			Created:       ts,
		},
		Networks:           networks,
		Pods:               pods,
		Services:           services,
		Ingresses:          ingresses,
		Status:             kl.Status,
		ReachableEndpoints: kl.ReachableEndpoints,
		CurrentStage:       kl.CurrentStage,
	}
}

func (kl *KubeLab) isReachable(epName string) bool {
	for _, b := range kl.ReachableEndpoints {
		if b == epName {
			return true
		}
	}
	return false
}

func (kl *KubeLab) setEndpointReachable(epName string) {
	// Return if already in slice
	if kl.isReachable(epName) {
		return
	}
	kl.ReachableEndpoints = append(kl.ReachableEndpoints, epName)
}

// ToLiveLesson exports a KubeLab as a generic LiveLesson so the API can use it
func (kl *KubeLab) ToLiveLesson() *pb.LiveLesson {

	ts, _ := strconv.ParseInt(kl.Namespace.ObjectMeta.Labels["created"], 10, 64)

	ret := pb.LiveLesson{
		LessonUUID:      kl.CreateRequest.Uuid,
		LessonId:        kl.CreateRequest.Lesson.LessonId,
		LiveEndpoints:   map[string]*pb.Endpoint{},
		LessonStage:     kl.CurrentStage,
		LessonDiagram:   kl.CreateRequest.Lesson.LessonDiagram,
		LessonVideo:     kl.CreateRequest.Lesson.LessonVideo,
		LabGuide:        kl.CreateRequest.Lesson.Stages[kl.CurrentStage].LabGuide,
		JupyterLabGuide: kl.CreateRequest.Lesson.Stages[kl.CurrentStage].JupyterLabGuide,
		CreatedTime: &timestamp.Timestamp{
			Seconds: ts,
		},
		LiveLessonStatus: kl.Status,
		TotalTests:       int32(kl.TotalTests),
		HealthyTests:     int32(kl.HealthyTests),
	}

	// Provide enriched (with Host and Port) Endpoint structs to API by using data from each Kubelab Service
	for s := range kl.Services {

		endpoint := &pb.Endpoint{}
		for i := range kl.CreateRequest.Lesson.Endpoints {
			ep := kl.CreateRequest.Lesson.Endpoints[i]
			if ep.Name == kl.Services[s].ObjectMeta.Name {
				endpoint.Name = ep.Name
				endpoint.ConfigurationType = ep.ConfigurationType
				endpoint.Host = kl.Services[s].Spec.ClusterIP
				endpoint.Presentations = ep.Presentations
				endpoint.AdditionalPorts = ep.AdditionalPorts
			}
		}
		ret.LiveEndpoints[endpoint.Name] = endpoint
	}

	return &ret
}

func (ls *LessonScheduler) createKubeLab(req *LessonScheduleRequest) (*KubeLab, error) {

	ns, err := ls.createNamespace(req)
	if err != nil {
		log.Error(err)
	}

	err = ls.syncSecret(ns.ObjectMeta.Name)
	if err != nil {
		log.Error("Unable to sync secret into this namespace. Ingress-based resources (like iframes) may not work.")
	}

	kl := &KubeLab{
		Namespace:     ns,
		CreateRequest: req,
		Networks:      map[string]*crd.NetworkAttachmentDefinition{},
		Pods:          map[string]*corev1.Pod{},
		Services:      map[string]*corev1.Service{},
		Ingresses:     map[string]*v1beta1.Ingress{},
	}

	// Append endpoint and create ingress for jupyter lab guide if necessary
	if usesJupyterLabGuide(req.Lesson) {
		jupyterEp := &pb.Endpoint{
			Name:            "jupyterlabguide",
			Image:           "antidotelabs/jupyter:newpath",
			AdditionalPorts: []int32{8888},
		}
		req.Lesson.Endpoints = append(req.Lesson.Endpoints, jupyterEp)

		iframeIngress, _ := ls.createIngress(
			ns.ObjectMeta.Name,
			jupyterEp,
			8888,
		)
		kl.Ingresses[iframeIngress.ObjectMeta.Name] = iframeIngress
	}

	// Create networks from connections property
	for c := range req.Lesson.Connections {
		connection := req.Lesson.Connections[c]
		newNet, err := ls.createNetwork(c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), req)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		// log.Infof("About to add %v at index %s", &newNet, &newNet.ObjectMeta.Name)

		kl.Networks[newNet.ObjectMeta.Name] = newNet
	}

	// Create pods and services
	for d := range req.Lesson.Endpoints {
		ep := req.Lesson.Endpoints[d]

		// Create pod
		newPod, err := ls.createPod(
			ep,
			getMemberNetworks(ep.Name, req.Lesson.Connections),
			req,
		)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		kl.Pods[newPod.ObjectMeta.Name] = newPod

		// Expose via service if needed
		if len(newPod.Spec.Containers[0].Ports) > 0 {
			newSvc, err := ls.createService(
				newPod,
				req,
			)
			if err != nil {
				log.Error(err)
				return nil, err
			}
			kl.Services[newSvc.ObjectMeta.Name] = newSvc
		}

		// Create appropriate presentations
		for pr := range ep.Presentations {
			p := ep.Presentations[pr]

			if p.Type == "http" {
				iframeIngress, _ := ls.createIngress(
					ns.ObjectMeta.Name,
					ep,
					p.Port,
				)
				kl.Ingresses[iframeIngress.ObjectMeta.Name] = iframeIngress
			} else if p.Type == "vnc" {
				// nothing to do?
			} else if p.Type == "ssh" {
				// nothing to do?
			}
		}
	}

	return kl, nil
}
