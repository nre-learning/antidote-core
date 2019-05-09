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
		LiveEndpoints:   map[string]*pb.LiveEndpoint{},
		LessonStage:     kl.CurrentStage,
		LessonDiagram:   kl.CreateRequest.Lesson.LessonDiagram,
		LessonVideo:     kl.CreateRequest.Lesson.LessonVideo,
		LabGuide:        kl.CreateRequest.Lesson.Stages[kl.CurrentStage].LabGuide,
		JupyterLabGuide: kl.CreateRequest.Lesson.Stages[kl.CurrentStage].JupyterLabGuide,
		CreatedTime: &timestamp.Timestamp{
			Seconds: ts,
		},
		LiveLessonStatus: kl.Status,
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

		endpoint := &pb.LiveEndpoint{
			Name: podBuddy.ObjectMeta.Name,
			Type: pb.LiveEndpoint_EndpointType(pb.LiveEndpoint_EndpointType_value[podBuddy.Labels["endpointType"]]),
			Host: host,
			Port: int32(port),
			// ApiPort
		}

		// Convert kubelab reachability to livelesson reachability
		if kl.isReachable(endpoint.Name) {
			endpoint.Reachable = true
		}

		ret.LiveEndpoints[endpoint.Name] = endpoint
	}

	for i := range kl.CreateRequest.Lesson.IframeResources {

		ifr := kl.CreateRequest.Lesson.IframeResources[i]

		endpoint := &pb.LiveEndpoint{
			Name:       ifr.Ref,
			Type:       pb.LiveEndpoint_IFRAME,
			IframePath: ifr.Path,
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

	// Append black box container and create ingress for jupyter lab guide if necessary
	if usesJupyterLabGuide(req.Lesson) {
		jupyterBB := &pb.Endpoint{
			Name:  "jupyterlabguide",
			Image: "antidotelabs/jupyter:newpath",
			Type:  pb.Endpoint_BLACKBOX,
			Ports: []int32{8888},
		}
		req.Lesson.Blackboxes = append(req.Lesson.Blackboxes, jupyterBB)

		iframeIngress, _ := ls.createIngress(
			ns.ObjectMeta.Name,
			&pb.IframeResource{
				Ref:      "jupyterlabguide",
				Protocol: "http",

				// Not needed. The front-end will append this specific path to the iframe src
				// Path:     fmt.Sprintf("/notebooks/lesson-%d/stage%d/notebook.ipynb", req.Lesson.LessonId, req.Stage),

				Port: 8888,
			},
		)
		kl.Ingresses[iframeIngress.ObjectMeta.Name] = iframeIngress
	}

	if HasDevices(kl.CreateRequest.Lesson) {

		log.Debug("Creating devices and connections")

		// Create networks from connections property
		for c := range req.Lesson.Connections {
			connection := req.Lesson.Connections[c]
			newNet, err := ls.createNetwork(c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), req)
			if err != nil {
				log.Error(err)
			}

			// log.Infof("About to add %v at index %s", &newNet, &newNet.ObjectMeta.Name)

			kl.Networks[newNet.ObjectMeta.Name] = newNet
		}

		// Create pods and services for devices
		for d := range req.Lesson.Devices {

			// Create pods from devices property
			device := req.Lesson.Devices[d]
			newPod, err := ls.createPod(
				device,
				getMemberNetworks(device.Name, req.Lesson.Connections),
				req,
			)
			if err != nil {
				log.Error(err)
				return nil, err
			}
			kl.Pods[newPod.ObjectMeta.Name] = newPod

			// Create service for this pod
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

	} else {
		log.Debug("Not creating devices and connections")
	}

	// Create pods and services for utility containers
	for d := range req.Lesson.Utilities {

		utility := req.Lesson.Utilities[d]
		newPod, err := ls.createPod(
			utility,
			getMemberNetworks(utility.Name, req.Lesson.Connections),
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
	for d := range req.Lesson.Blackboxes {

		blackbox := req.Lesson.Blackboxes[d]
		newPod, err := ls.createPod(
			blackbox,
			getMemberNetworks(blackbox.Name, req.Lesson.Connections),
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

	// Create pods, services, and ingresses for iframe resources
	for d := range req.Lesson.IframeResources {

		ifr := req.Lesson.IframeResources[d]

		// Iframe resources don't create pods/services on their own. You must define a blackbox/utility/device endpoint
		// and then refer to that in the iframeresource definition. We're just creating an ingress here to access that endpoint.

		iframeIngress, _ := ls.createIngress(
			ns.ObjectMeta.Name,
			ifr,
		)
		kl.Ingresses[iframeIngress.ObjectMeta.Name] = iframeIngress

	}
	return kl, nil
}
