package scheduler

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LessonScheduler) deleteService(name string) error {
	return nil
}

func (ls *LessonScheduler) createService(pod *corev1.Pod, req *LessonScheduleRequest) (*corev1.Service, error) {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	// We want to use the same name as the Pod object, since the service name will be what users try to reach
	// (i.e. use "vqfx1" instead of "vqfx1-svc" or something like that.)
	serviceName := pod.ObjectMeta.Name

	nsName := fmt.Sprintf("%d-%s-ns", req.LessonDef.LessonID, req.Session)

	serviceTypeMap := map[string]corev1.ServiceType{
		"DEVICE":   corev1.ServiceTypeClusterIP,
		"NOTEBOOK": corev1.ServiceTypeLoadBalancer,
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":         fmt.Sprintf("%d", req.LessonDef.LessonID),
				"lessonInstanceId": req.Session,
				"syringeManaged":   "yes",
				"endpointType":     pod.ObjectMeta.Labels["endpointType"],
				"sshUser":          pod.ObjectMeta.Labels["sshUser"],
				"sshPassword":      pod.ObjectMeta.Labels["sshPassword"],
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"lessonId":  fmt.Sprintf("%d", req.LessonDef.LessonID),
				"sessionId": req.Session,
				"podName":   pod.ObjectMeta.Name,
			},
			Ports: []corev1.ServicePort{}, // will fill out below

			Type: serviceTypeMap[pod.ObjectMeta.Labels["endpointType"]],
		},
	}

	for p := range pod.Spec.Containers[0].Ports {

		// TODO(mierdin): need to do something different for NOTEBOOK/IFRAME endpoints

		port := pod.Spec.Containers[0].Ports[p].ContainerPort

		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
			Name:       fmt.Sprintf("port-%d", port),
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
		})
	}

	result, err := coreclient.Services(nsName).Create(svc)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created service: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Service %s already exists.", serviceName)
		result, err := coreclient.Services(nsName).Get(serviceName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve service after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Error creating service: %s", err)
		return nil, err
	}

	return result, err
}
