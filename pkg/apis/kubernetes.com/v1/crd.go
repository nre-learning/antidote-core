/*

Code modified from https://github.com/yaronha/kube-crd

*/

package v1

import (
	"reflect"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

const (
	CRDPlural   string = "networks"
	CRDGroup    string = "kubernetes.com"
	CRDVersion  string = "v1"
	FullCRDName string = CRDPlural + "." + CRDGroup
)

// Create the CRD resource, ignore error if it already exists
func CreateCRD(clientset apiextcs.Interface) error {
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{Name: FullCRDName},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   CRDGroup,
			Version: CRDVersion,
			Scope:   apiextv1beta1.NamespaceScoped,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural: CRDPlural,
				Kind:   reflect.TypeOf(Network{}).Name(),
			},
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err

	// Note the original apiextensions example adds logic to wait for creation and exception handling
}

// https://www.martin-helmich.de/en/blog/kubernetes-crd-client.html
// https://github.com/yaronha/kube-crd/blob/master/crd/crd.go#L16:1
// apiVersion: "kubernetes.com/v1"
// kind: Network
// metadata:
//   name: flannel-networkobj
// plugin: flannel
// args: '[
// 		{
// 				"delegate": {
// 						"isDefaultGateway": true
// 				}
// 		}
// ]'

// apiVersion: apiextensions.k8s.io/v1beta1
// kind: CustomResourceDefinition
// metadata:
//   name: networks.kubernetes.com
// spec:
//   group: kubernetes.com
//   version: v1
//   scope: Namespaced
//   names:
//     plural: networks
//     singular: network
//     kind: Network
//     shortNames:
//     - net

// https://github.com/yaronha/kube-crd

// https://github.com/openshift-evangelists/crd-code-generation
// https://kubernetes.io/blog/2018/01/introducing-client-go-version-6/
// https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

// Create a  Rest client with the new CRD Schema
// var SchemeGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}

// func addKnownTypes(scheme *runtime.Scheme) error {
// 	scheme.AddKnownTypes(SchemeGroupVersion,
// 		&Network{},
// 		&NetworkList{},
// 	)
// 	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
// 	return nil
// }

func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	config := *cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}
	return client, scheme, nil
}
