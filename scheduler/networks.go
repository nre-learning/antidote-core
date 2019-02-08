package scheduler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	crd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/nre-learning/syringe/pkg/client"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	netv1client "k8s.io/client-go/kubernetes/typed/networking/v1"
)

func (ls *LessonScheduler) createNetworkCrd() error {

	// create clientset and create our CRD, this only need to run once
	clientset, err := apiextcs.NewForConfig(ls.KubeConfig)
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

// createNetworkPolicy applies a kubernetes networkpolicy object to prohibit traffic out of the created namespace, for all
// pods that aren't used for configuration purposes.
func (ls *LessonScheduler) createNetworkPolicy(nsName string) (*netv1.NetworkPolicy, error) {

	nc, err := netv1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	var tcp corev1.Protocol = "TCP"
	var udp corev1.Protocol = "UDP"
	fivethree := intstr.IntOrString{IntVal: 53}

	np := netv1.NetworkPolicy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "stoneage",
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
			},
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: meta_v1.LabelSelector{
				MatchExpressions: []meta_v1.LabelSelectorRequirement{
					{
						Key:      "syringeManaged",
						Operator: meta_v1.LabelSelectorOpIn,
						Values: []string{
							"yes",
						},
					},
					{ // do not apply network policy to config pods, they need to get to internet for configs
						Key:      "configPod",
						Operator: meta_v1.LabelSelectorOpNotIn,
						Values: []string{
							"yes",
						},
					},
					{ // do not apply network policy to verify pods, they need to get to internet for configs
						Key:      "verifyPod",
						Operator: meta_v1.LabelSelectorOpNotIn,
						Values: []string{
							"yes",
						},
					},
				},
			},
			PolicyTypes: []netv1.PolicyType{
				// Apply only egress policy here. We want to permit all ingress traffic,
				// so that Syringe and Guacamole can reach these endpoints unhindered.
				//
				// We really only care about restricting what the pods can access, especially
				// the internet.
				"Egress",
			},
			Egress: []netv1.NetworkPolicyEgressRule{

				// Allow only intra-namespace traffic
				{
					To: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &meta_v1.LabelSelector{MatchLabels: map[string]string{
								"name": nsName,
							}},
						},
					},
				},
				// Allow traffic to the cluster's DNS service
				{
					To: []netv1.NetworkPolicyPeer{

						// Have only been able to get this working with this CIDR.
						// Tried a /32 directly to the svc IP, but that didn't seem to work.
						// Should revisit this later.
						{IPBlock: &netv1.IPBlock{CIDR: "10.0.0.0/8"}},
					},
					Ports: []netv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &fivethree},
						{Protocol: &udp, Port: &fivethree},
					},
				},
			},
		},
	}

	newnp, err := nc.NetworkPolicies(nsName).Create(&np)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Info("Created networkpolicy")

	} else if apierrors.IsAlreadyExists(err) {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Warn("networkpolicy already exists.")
		return newnp, nil
	} else {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Errorf("Problem creating networkpolicy: %s", err)

		return nil, err
	}
	return newnp, nil

}

func (ls *LessonScheduler) createNetwork(netIndex int, netName string, req *LessonScheduleRequest, deviceNetwork bool, subnet string) (*crd.NetworkAttachmentDefinition, error) {

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	nsName := fmt.Sprintf("%s-ns", req.Uuid)

	// Create a CRD client interface
	crdclient := client.CrdClient(crdcs, scheme, nsName)

	networkName := fmt.Sprintf("%s-%s", nsName, netName)

	// Max of 15 characters in the bridge name - https://access.redhat.com/solutions/652593
	bridgeName := fmt.Sprintf("%d-%s", netIndex, req.Uuid)
	if len(bridgeName) > 15 {
		bridgeName = bridgeName[0:15]
	}

	networkArgs := fmt.Sprintf(`{
			"name": "%s",
			"type": "antibridge",
			"plugin": "antibridge",
			"bridge": "%s",
			"forceAddress": false,
			"hairpinMode": false,
			"delegate": {
					"hairpinMode": false
			},
			"ipam": {
			  "type": "host-local",
			  "subnet": "%s"
			}
		}`, networkName, bridgeName, subnet)

	// Create a new Network object and write to k8s
	network := &crd.NetworkAttachmentDefinition{
		// apiVersion: "k8s.cni.cncf.io/v1",
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      netName,
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonId),
				"syringeManaged": "yes",
			},
		},
		Kind: "NetworkAttachmentDefinition",
		Spec: crd.NetworkSpec{
			Config: networkArgs,
		},
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
func getMemberNetworks(deviceName string, connections []*pb.Connection) []string {
	// We want the management network to be first always.
	// EDIT: Commented out since the management network is provided implicitly for now. We may want to move to an explicit model soon.
	// memberNets := []string{
	// 	"mgmt-net",
	// }
	memberNets := []string{}
	for c := range connections {
		connection := connections[c]
		if connection.A == deviceName || connection.B == deviceName {
			netName := fmt.Sprintf("%s-%s-net", connection.A, connection.B)
			memberNets = append(memberNets, netName)
		}
	}
	return memberNets
}

func (ls *LessonScheduler) deleteNetwork(name, ns string) error {

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(ls.KubeConfig)
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
