package labs

import (
	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LabScheduler) deleteService(name string) error {

	// // Create a new clientset which include our CRD schema
	// crdcs, scheme, err := crd.NewClient(ls.Config)
	// if err != nil {
	// 	panic(err)
	// }

	// // Create a CRD client interface
	// crdclient := client.CrdClient(crdcs, scheme, "default")

	// err = crdclient.Delete(name, &meta_v1.DeleteOptions{})
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (ls *LabScheduler) createService(name, labId, labInstanceId string) (*corev1.Service, error) {

	coreclient, err := corev1client.NewForConfig(ls.Config)
	if err != nil {
		panic(err)
	}
	serviceName := labId + labInstanceId + "svc" + name
	podName := labId + labInstanceId + "pod" + name

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			Labels: map[string]string{
				"labId":          labId,
				"labInstanceId":  labInstanceId,
				"syringeManaged": "yes",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"labId":         labId,
				"labInstanceId": labInstanceId,
				"podName":       podName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "ssh",
					Port:       22,
					TargetPort: intstr.FromInt(22),
				},
				{
					Name:       "netconf",
					Port:       830,
					TargetPort: intstr.FromInt(830),
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	result, err := coreclient.Services("default").Create(svc)
	if err == nil {
		log.Infof("Created service: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Service %s already exists.", serviceName)
		result, err := coreclient.Services("default").Get(serviceName, metav1.GetOptions{})
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
