package scheduler

import (
	"fmt"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"

	// Custom Network CRD Types
	networkcrd "github.com/nre-learning/antidote-core/pkg/apis/k8s.cni.cncf.io/v1"

	// Kubernetes Types
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func (s *AntidoteScheduler) createNetworkCrd() error {

	// NOTE: if the CRD already exists, this function will return with no error. The idea is
	// that we run this every time Antidote starts just to make sure our CRD is installed in the cluster.
	//
	// Note the import path - this is code that we control, so if we desire a different
	// outcome, it's possible. Just be aware this function is called early, so changes in behavior will be
	// noticed.
	err := networkcrd.CreateCRD(s.ClientExt)
	if err != nil {
		return err
	}
	return nil
}

// createNetworkPolicy applies a kubernetes networkpolicy object control traffic out of the namespace.
// The main use case is to restrict access for lesson users to only resources in that lesson,
// with some exceptions.
func (s *AntidoteScheduler) createNetworkPolicy(sc ot.SpanContext, nsName string) (*netv1.NetworkPolicy, error) {
	span := ot.StartSpan("scheduler_networkpolicy_create", ot.ChildOf(sc))
	defer span.Finish()

	var tcp corev1.Protocol = "TCP"
	var udp corev1.Protocol = "UDP"
	fivethree := intstr.IntOrString{IntVal: 53}

	np := netv1.NetworkPolicy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "stoneage",
			Namespace: nsName,
			Labels: map[string]string{
				"antidoteManaged": "yes",
			},
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: meta_v1.LabelSelector{
				MatchExpressions: []meta_v1.LabelSelectorRequirement{
					{
						Key:      "antidoteManaged",
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
				// We really only care about restricting what the pods can access, especially
				// the internet. Everything else should be able to access into the namespace
				// unrestricted. Thus, applying only in the egress direction.
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
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}
	return newnp, nil

}

// createNetwork
func (s *AntidoteScheduler) createNetwork(sc ot.SpanContext, netIndex int, netName string, req services.LessonScheduleRequest) (*networkcrd.NetworkAttachmentDefinition, error) {
	span := ot.StartSpan("scheduler_network_create", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	networkName := fmt.Sprintf("%s-%s", nsName, netName)

	// Max of 15 characters in the bridge name - https://access.redhat.com/solutions/652593
	livelesson := req.LiveLessonID
	if len(livelesson) > 6 {
		livelesson = livelesson[0:6]
	}
	antidoteId := s.Config.InstanceID
	if len(antidoteId) > 6 {
		antidoteId = antidoteId[0:6]
	}
	// Combined, these use no more than 12 characters. This leaves three digits for the netIndex, which
	// should be way more than enough.
	bridgeName := fmt.Sprintf("%d%s%s", netIndex, antidoteId, livelesson)

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
				"antidoteManaged": "yes",
			},
		},
		Kind: "NetworkAttachmentDefinition",
		Spec: networkcrd.NetworkSpec{
			Config: networkArgs,
		},
	}

	nadClient := s.ClientCrd.K8s().NetworkAttachmentDefinitions(nsName)

	result, err := nadClient.Create(network)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
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
