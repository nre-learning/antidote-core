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

func (ls *LabScheduler) deleteService(name string) error {
	return nil
}

func (ls *LabScheduler) createService(pod *corev1.Pod, req *LabScheduleRequest) (*corev1.Service, error) {

	coreclient, err := corev1client.NewForConfig(ls.Config)
	if err != nil {
		panic(err)
	}
	serviceName := pod.ObjectMeta.Name + "-svc"

	nsName := fmt.Sprintf("%d-%s-ns", req.LabDef.LabID, req.Session)

	typePortMap := map[string]int32{
		"DEVICE":   22,
		"NOTEBOOK": 8888,
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nsName,
			Labels: map[string]string{
				"labId":          fmt.Sprintf("%d", req.LabDef.LabID),
				"labInstanceId":  req.Session,
				"syringeManaged": "yes",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"labId":     fmt.Sprintf("%d", req.LabDef.LabID),
				"sessionId": req.Session,
				"podName":   pod.ObjectMeta.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "primaryport",
					Port:       typePortMap[pod.ObjectMeta.Labels["endpointType"]],
					TargetPort: intstr.FromInt(int(typePortMap[pod.ObjectMeta.Labels["endpointType"]])),
				},
				// Not currently used, will be used soon
				// {
				// 	Name:       "apiPort",
				// 	Port:       830,
				// 	TargetPort: intstr.FromInt(830),
				// },
			},
			Type: corev1.ServiceTypeNodePort,
		},
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
