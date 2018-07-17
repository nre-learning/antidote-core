// Responsible for creating all resources for a lab. Pods, services, networks, etc.
package labs

import (
	"errors"
	"strconv"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type LabScheduler struct {
	Config *rest.Config
}

// returning a single *Lab to test the API for now. Should be a channel in the future, and this launched as a goroutine
func (ls *LabScheduler) Start() ([]*Lab, error) {

	lab1, err := ls.createLab()
	if err != nil {
		log.Errorf("Error creating lab: %s", err)
		return nil, err
	}
	log.Infof("Created lab:\n%+v\n", lab1)
	log.Infof(lab1.LabConnections["csrx1"])
	// lab1.TearDown()

	return []*Lab{
		lab1,
	}, nil

	// for {
	// 	log.Info("Sleeping....")
	// 	time.Sleep(1000)
	// }
}

func (ls *LabScheduler) createLab() (*Lab, error) {
	ls.createNetworkCrd()
	network1, _ := ls.createNetwork("lab001", "lab001-0001")

	pod1, _ := ls.createPod("csrx1", "lab001", "lab001-0001")
	log.Info("Creating service: csrx1")
	service1, _ := ls.createService("csrx1svc", "lab001", "lab001-0001")

	service1Port, _ := getSSHServicePort(service1)

	return &Lab{
		Networks: []string{
			network1.ObjectMeta.Name,
		},
		Pods: []string{
			pod1.ObjectMeta.Name,
		},
		Services: []string{
			service1.ObjectMeta.Name,
		},
		LabConnections: map[string]string{
			"csrx1": service1Port,
		},
	}, nil
}

func getSSHServicePort(svc *corev1.Service) (string, error) {
	for p := range svc.Spec.Ports {
		if svc.Spec.Ports[p].Port == 22 {

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

type Lab struct {
	Networks       []string
	Pods           []string
	Services       []string
	LabConnections map[string]string
}

func (ls *LabScheduler) TearDown(l *Lab) error {

	for n := range l.Services {
		ls.deleteService(l.Services[n])
	}

	for n := range l.Pods {
		ls.deletePod(l.Pods[n])
	}

	for n := range l.Networks {
		ls.deleteNetwork(l.Networks[n])
	}

	return nil

}
