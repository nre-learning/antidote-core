package main

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	log "github.com/Sirupsen/logrus"
	"github.com/nre-learning/syringe/labs"

	api "github.com/nre-learning/syringe/api/exp"

	"k8s.io/client-go/rest"
)

// return rest config, if path not specified assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func main() {

	/*

	   Users need to provide a lab definition which shows which containers connecting in which ways, and how many copies syringe should maintain

	   - Provision namespace
	   - Provision virtual networks needed by the lab
	   - Provision pods and services

	*/

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := GetClientConfig(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	labScheduler := labs.LabScheduler{
		Config: config,
	}
	activeLabs, err := labScheduler.Start()
	if err != nil {
		log.Errorf("Problem starting lab scheduler: %s", err)
	}

	go api.StartAPI(activeLabs)

	c := make(chan struct{})
	<-c
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
