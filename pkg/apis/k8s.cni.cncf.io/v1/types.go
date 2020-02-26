package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// gentypes "k8s.io/code-generator/code-generator/cmd/client-gen/types"
)

// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/generating-clientset.md
// https://www.martin-helmich.de/en/blog/kubernetes-crd-client.html
// https://github.com/yaronha/kube-crd/blob/master/crd/crd.go#L16:1
// https://github.com/yaronha/kube-crd
// https://github.com/openshift-evangelists/crd-code-generation
// https://kubernetes.io/blog/2018/01/introducing-client-go-version-6/
// https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

// MULTUS V3
//
// ---
// apiVersion: apiextensions.k8s.io/v1beta1
// kind: CustomResourceDefinition
// metadata:
//   name: network-attachment-definitions.k8s.cni.cncf.io
// spec:
//   group: k8s.cni.cncf.io
//   version: v1
//   scope: Namespaced
//   names:
//     plural: network-attachment-definitions
//     singular: network-attachment-definition
//     kind: NetworkAttachmentDefinition
//     shortNames:
//     - net-attach-def
//   validation:
//     openAPIV3Schema:
//       properties:
//         spec:
//           properties:
//             config:
//                  type: string

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkAttachmentDefinition struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               NetworkSpec `json:"spec"`
	Args               string      `json:"args"`
	Kind               string      `json:"kind"`
	// foobar             gentypes.Version
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkAttachmentDefinitionList struct {
	meta_v1.TypeMeta `json:",inline"`
	// +optional
	meta_v1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkAttachmentDefinition `json:"items"`
}

type NetworkSpec struct {
	Group      string            `json:"group"`
	Version    string            `json:"version"`
	Scope      string            `json:"scope,omitempty"`
	Names      NetworkNames      `json:"names,omitempty"`
	Config     string            `json:"config"`
	Validation NetworkValidation `json:"validation"`
}
type NetworkNames struct {
	Plural     string   `json:"networks,omitempty"`
	Singular   string   `json:"singular,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	ShortNames []string `json:"shortNames,omitempty"`
}

type NetworkValidation struct {
	OpenAPIV3Schema NetworkValidationSchema `json:"openAPIV3Schema"`
}

type NetworkValidationSchema struct {
	Properties NetworkValidationProperties `json:"properties"`
}

type NetworkValidationProperties struct {
	Spec NetworkValidationSpec `json:"spec"`
}

type NetworkValidationSpec struct {
	Properties NetworkValidationSpecProperties `json:"properties"`
}

type NetworkValidationSpecProperties struct {
	Config NetworkValidationConfig `json:"config"`
}

type NetworkValidationConfig struct {
	Type string `json:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []NetworkAttachmentDefinition `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=github.com/nre-learning/antidote-core/pkg/apis/k8s.cni.cncf.io/v1/v1.K8sV1Interfacez
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// type K8sV1Interface struct{}
// type K8sV1Client struct{}
