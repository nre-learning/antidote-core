package v1

import meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Network struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               NetworkSpec `json:"spec"`
	Args               string      `json:"args"`
	Kind               string      `json:"kind"`
}
type NetworkSpec struct {
	Group   string       `json:"group"`
	Version string       `json:"version"`
	Scope   string       `json:"scope,omitempty"`
	Names   NetworkNames `json:"names,omitempty"`
}
type NetworkNames struct {
	Plural     string   `json:"networks,omitempty"`
	Singular   string   `json:"singular,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	ShortNames []string `json:"shortNames,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []Network `json:"items"`
}
