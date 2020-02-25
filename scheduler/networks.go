package scheduler

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	models "github.com/nre-learning/syringe/db/models"
	"github.com/nre-learning/syringe/services"

	// Custom Network CRD Types
	networkcrd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"

	// Kubernetes Types
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func (s *AntidoteScheduler) createNetworkCrd() error {

	// note: if the CRD exist our CreateCRD function is set to exit without an error
	err := networkcrd.CreateCRD(s.ClientExt)
	if err != nil {
		panic(err) // TODO(mierdin): boooooo. Get rid of this
	}

	// Wait for the CRD to be created before we use it (only needed if its a new one)
	// TODO(mierdin): This really shouldn't be necessary. Let's try removing it.
	// time.Sleep(3 * time.Second)

	return nil
}

// createNetworkPolicy applies a kubernetes networkpolicy object control traffic out of the namespace.
// The main use case is to restrict access for lesson users to only resources in that lesson,
// with some exceptions.
func (s *AntidoteScheduler) createNetworkPolicy(nsName string) (*netv1.NetworkPolicy, error) {

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
						// Tried a /32 directly to the svc IP for DNS, but that didn't seem to work.
						// Should revisit this later. Open to all RFC1918 for now.
						{IPBlock: &netv1.IPBlock{CIDR: "10.0.0.0/8"}},
						{IPBlock: &netv1.IPBlock{CIDR: "192.168.0.0/16"}},
						{IPBlock: &netv1.IPBlock{CIDR: "171.16.0.0/12"}},
					},
					Ports: []netv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &fivethree},
						{Protocol: &udp, Port: &fivethree},
					},
				},
			},
		},
	}

	newnp, err := s.Client.NetworkingV1().NetworkPolicies(nsName).Create(&np)
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

// createNetwork
func (s *AntidoteScheduler) createNetwork(netIndex int, netName string, req services.LessonScheduleRequest) (*networkcrd.NetworkAttachmentDefinition, error) {
	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	networkName := fmt.Sprintf("%s-%s", nsName, netName)

	// Max of 15 characters in the bridge name - https://access.redhat.com/solutions/652593
	livelesson := req.LiveLessonID
	if len(livelesson) > 6 {
		livelesson = livelesson[0:6]
	}
	syringeID := s.Config.InstanceID
	if len(syringeID) > 6 {
		syringeID = syringeID[0:6]
	}
	// Combined, these use no more than 12 characters. This leaves three digits for the netIndex, which
	// should be way more than enough.
	bridgeName := fmt.Sprintf("%d%s%s", netIndex, syringeID, livelesson)

	// NOTE that this is just a placeholder, not necessarily the actual subnet in use on this segment.
	// We have to put SOMETHING here, but because we're using the bridge plugin, this isn't actually
	// enforced, which is desired behaviors. Endpoints can still use their own subnets.
	subnet := "10.10.0.0/16"

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

	network := &networkcrd.NetworkAttachmentDefinition{
		// apiVersion: "k8s.cni.cncf.io/v1",
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      netName,
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
			},
		},
		Kind: "NetworkAttachmentDefinition",
		Spec: networkcrd.NetworkSpec{
			Config: networkArgs,
		},
	}

	nadClient := s.ClientCrd.K8s().NetworkAttachmentDefinitions(nsName)

	result, err := nadClient.Create(network)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created network: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Network %s already exists.", network.ObjectMeta.Name)

		result, err := nadClient.Get(network.ObjectMeta.Name, meta_v1.GetOptions{})
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

// getMemberNetworks gets the names of all networks an endpoint belongs to based on definition.
func getMemberNetworks(epName string, connections []*models.LessonConnection) []string {
	// We want the management network to be first always.
	// EDIT: Commented out since the management network is provided implicitly for now. We may want to move to an explicit model soon.
	// memberNets := []string{
	// 	"mgmt-net",
	// }
	memberNets := []string{}
	for c := range connections {
		connection := connections[c]
		if connection.A == epName || connection.B == epName {
			netName := fmt.Sprintf("%s-%s-net", connection.A, connection.B)
			memberNets = append(memberNets, netName)
		}
	}
	return memberNets
}
