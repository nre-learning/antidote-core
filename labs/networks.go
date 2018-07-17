package labs

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"

	crd "github.com/nre-learning/syringe/pkg/apis/kubernetes.com/v1"
	"github.com/nre-learning/syringe/pkg/client"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ls *LabScheduler) createNetworkCrd() error {

	// create clientset and create our CRD, this only need to run once
	clientset, err := apiextcs.NewForConfig(ls.Config)
	if err != nil {
		panic(err.Error())
	}

	// note: if the CRD exist our CreateCRD function is set to exit without an error
	err = crd.CreateCRD(clientset)
	if err != nil {
		panic(err)
	}

	// Wait for the CRD to be created before we use it (only needed if its a new one)
	time.Sleep(3 * time.Second)

	return nil
}

func (ls *LabScheduler) createNetwork(labId, labInstanceId string) (*crd.Network, error) {
	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(ls.Config)
	if err != nil {
		panic(err)
	}

	// Create a CRD client interface
	crdclient := client.CrdClient(crdcs, scheme, "default")

	netName := labId + labInstanceId + "net"

	// Create a new Network object and write to k8s
	network := &crd.Network{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: netName,
			Labels: map[string]string{
				"labId":          labId,
				"labInstanceId":  labInstanceId,
				"syringeManaged": "yes",
			},
		},
		Kind: "Network",
		Args: fmt.Sprintf("[ { 'name': '%s', 'type': 'weave-net', 'hairpinMode': false, 'delegate': { 'hairpinMode': false } } ]", netName),
	}

	result, err := crdclient.Create(network)
	if err == nil {
		log.Infof("Created network: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Network %s already exists.", network.ObjectMeta.Name)

		// In this case we are returning what we tried to create. This means that when this lab is cleaned up,
		// syringe will delete the network that already existed.
		return network, err
	} else {
		return nil, err
	}
	return result, err
}

func (ls *LabScheduler) deleteNetwork(name string) error {

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(ls.Config)
	if err != nil {
		panic(err)
	}

	// Create a CRD client interface
	crdclient := client.CrdClient(crdcs, scheme, "default")

	err = crdclient.Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
