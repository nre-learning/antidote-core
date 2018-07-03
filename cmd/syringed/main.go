package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"time"

	crd "github.com/nre-learning/syringe/pkg/apis/kubernetes.com/v1"
	"github.com/nre-learning/syringe/pkg/client"

	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// return rest config, if path not specified assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func newCrdFunc() {
	kubeconf := flag.String("kubeconf", "admin.conf", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	config, err := GetClientConfig(*kubeconf)
	if err != nil {
		panic(err.Error())
	}

	// create clientset and create our CRD, this only need to run once
	clientset, err := apiextcs.NewForConfig(config)
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

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := crd.NewClient(config)
	if err != nil {
		panic(err)
	}

	// Create a CRD client interface
	crdclient := client.CrdClient(crdcs, scheme, "default")

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

	// Create a new Example object and write to k8s
	example := &crd.Network{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   "networks.kubernetes.com",
			Labels: map[string]string{"mylabel": "test"},
		},
		Spec: crd.NetworkSpec{
			Group:   "kubernetes.com",
			Version: "v1",
			Scope:   "Namespaced",
			Names: crd.NetworkNames{
				Plural:   "networks",
				Singular: "network",
				Kind:     "Network",
				ShortNames: []string{
					"net",
				},
			},
		},
	}

	result, err := crdclient.Create(example)
	if err == nil {
		fmt.Printf("CREATED: %#v\n", result)
	} else if apierrors.IsAlreadyExists(err) {
		fmt.Printf("ALREADY EXISTS: %#v\n", result)
	} else {
		panic(err)
	}

	// List all Example objects
	items, err := crdclient.List(meta_v1.ListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("List:\n%s\n", items)

	// // Example Controller
	// // Watch for changes in Example objects and fire Add, Delete, Update callbacks
	// _, controller := cache.NewInformer(
	// 	crdclient.NewListWatch(),
	// 	&crd.Example{},
	// 	time.Minute*10,
	// 	cache.ResourceEventHandlerFuncs{
	// 		AddFunc: func(obj interface{}) {
	// 			fmt.Printf("add: %s \n", obj)
	// 		},
	// 		DeleteFunc: func(obj interface{}) {
	// 			fmt.Printf("delete: %s \n", obj)
	// 		},
	// 		UpdateFunc: func(oldObj, newObj interface{}) {
	// 			fmt.Printf("Update old: %s \n      New: %s\n", oldObj, newObj)
	// 		},
	// 	},
	// )

	// stop := make(chan struct{})
	// go controller.Run(stop)

	// // Wait forever
	// select {}
}

func main() {

	/*

			DOCS:
			https://godoc.org/k8s.io/client-go

		   Users need to provide a lab definition which shows which containers connecting in which ways, and how many copies syringe should maintain

		   - Provision namespace
		   - Provision virtual networks needed by the lab
		   - Provision pods and services

	*/

	// var kubeconfig *string
	// if home := homeDir(); home != "" {
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	// } else {
	// 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	// }
	// flag.Parse()

	// // use the current context in kubeconfig
	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// // create the clientset
	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// existingNamespaces, err := clientset.Core().Namespaces().List(metav1.ListOptions{})
	// if err != nil {
	// 	panic(err.Error())
	// }
	// fmt.Printf("EXISTING NAMESPACES: %s", existingNamespaces)

	// nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "antidote-lesson1-abcdef"}}
	// _, err = clientset.Core().Namespaces().Create(nsSpec)

	newCrdFunc()

	// for {
	// 	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	// 	if err != nil {
	// 		panic(err.Error())
	// 	}
	// 	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	// 	// Examples for error handling:
	// 	// - Use helper functions like e.g. errors.IsNotFound()
	// 	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	// 	namespace := "default"
	// 	pod := "example-xxxxx"
	// 	_, err = clientset.CoreV1().Pods(namespace).Get(pod, metav1.GetOptions{})
	// 	if errors.IsNotFound(err) {
	// 		fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
	// 	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
	// 		fmt.Printf("Error getting pod %s in namespace %s: %v\n",
	// 			pod, namespace, statusError.ErrStatus.Message)
	// 	} else if err != nil {
	// 		panic(err.Error())
	// 	} else {
	// 		fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
	// 	}

	// 	time.Sleep(10 * time.Second)
	// }
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
