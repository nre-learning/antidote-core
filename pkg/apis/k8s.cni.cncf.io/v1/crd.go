/*

Code modified from https://github.com/yaronha/kube-crd

*/

package v1

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cloudflare/cfssl/log"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

const (
	CRDPlural   string = "network-attachment-definitions"
	CRDGroup    string = "k8s.cni.cncf.io"
	CRDVersion  string = "v1"
	FullCRDName string = CRDPlural + "." + CRDGroup
)

// Create the CRD resource, ignore error if it already exists
func CreateCRD(clientset apiextcs.Interface) error {

	// Had to do this silliness because apiextv1beta1 needed it's own vendored ObjectMeta instead of my v1.ObjectMeta for some dumb reason
	str := fmt.Sprintf(`{
		"metadata": {
			"name": %s
		}
	}`, FullCRDName)
	crd := apiextv1beta1.CustomResourceDefinition{}
	json.Unmarshal([]byte(str), &crd)

	log.Debugf("Created unmarshaled CRD: %v", crd)

	crd.Spec = apiextv1beta1.CustomResourceDefinitionSpec{
		Group:   CRDGroup,
		Version: CRDVersion,
		Scope:   apiextv1beta1.NamespaceScoped,
		Names: apiextv1beta1.CustomResourceDefinitionNames{
			Plural: CRDPlural,
			Kind:   reflect.TypeOf(NetworkAttachmentDefinition{}).Name(),
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&crd)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err

	// Note the original apiextensions example adds logic to wait for creation and exception handling
}

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
