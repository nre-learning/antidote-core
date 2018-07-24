package scheduler

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/nre-learning/syringe/def"
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

func (ls *LabScheduler) createNetwork(netName string, req *LabScheduleRequest) (*crd.Network, error) {

	// type Connection struct {
	// 	A string `json:"a" yaml:"a"`
	// 	B string `json:"b" yaml:"b"`
	// }

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(ls.Config)
	if err != nil {
		panic(err)
	}

	nsName := fmt.Sprintf("%d-%s-ns", req.LabDef.LabID, req.Session)

	// Create a CRD client interface
	crdclient := client.CrdClient(crdcs, scheme, nsName)

	// Create a new Network object and write to k8s
	network := &crd.Network{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      netName,
			Namespace: nsName,
			Labels: map[string]string{
				"labId":          fmt.Sprintf("%d", req.LabDef.LabID),
				"sessionId":      req.Session,
				"syringeManaged": "yes",
			},
		},
		Kind: "Network",
		Args: fmt.Sprintf("[ { 'name': '%s', 'type': 'weave-net', 'hairpinMode': false, 'delegate': { 'hairpinMode': false } } ]", netName),
	}

	result, err := crdclient.Create(network)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created network: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Network %s already exists.", network.ObjectMeta.Name)

		result, err := crdclient.Get(network.ObjectMeta.Name)
		if err != nil {
			log.Errorf("Couldn't retrieve network after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating network %s: %s", netName, err)
		return nil, err
	}
	return result, err
}

// getMemberNetworks gets the names of all networks a device belongs to based on definition.
func getMemberNetworks(device *def.Device, connections []*def.Connection) []string {
	// We want the management network to be first always.
	memberNets := []string{
		"mgmt-net",
	}
	for c := range connections {
		connection := connections[c]
		if connection.A == device.Name || connection.B == device.Name {
			netName := fmt.Sprintf("%s-%s-net", connection.A, connection.B)
			memberNets = append(memberNets, netName)
		}
	}
	return memberNets
}

func (ls *LabScheduler) deleteNetwork(name, ns string) error {

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(ls.Config)
	if err != nil {
		panic(err)
	}

	// Create a CRD client interface
	crdclient := client.CrdClient(crdcs, scheme, ns)

	err = crdclient.Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
