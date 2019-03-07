package main

import (
	"fmt"
	"io/ioutil"
	"sync"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	api "github.com/nre-learning/syringe/api/exp"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	"github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	crd "github.com/nre-learning/syringe/pkg/apis/k8s.cni.cncf.io/v1"
	crdclient "github.com/nre-learning/syringe/pkg/client"

	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {

	syringeConfig, err := config.LoadConfigVars()
	if err != nil {
		log.Error(err)
		log.Fatalf("Invalid configuration. Please re-run Syringe with appropriate env variables")
	}

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	lessonDefs, err := api.ImportLessonDefs(syringeConfig, syringeConfig.LessonsDir)
	if err != nil {
		log.Warn(err)
	}

	// Start lesson scheduler
	lessonScheduler := scheduler.LessonScheduler{
		KubeConfig:    kubeConfig,
		Requests:      make(chan *scheduler.LessonScheduleRequest),
		Results:       make(chan *scheduler.LessonScheduleResult),
		LessonDefs:    lessonDefs,
		SyringeConfig: syringeConfig,
		GcWhiteList:   make(map[string]*pb.Session),
		GcWhiteListMu: &sync.Mutex{},
		KubeLabs:      make(map[string]*scheduler.KubeLab),
		KubeLabsMu:    &sync.Mutex{},
	}

	// CREATION OF CLIENTS
	//
	// Client for working with standard kubernetes resources
	cs, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Error(err)
		log.Fatalf("Invalid kubeconfig")
	}
	lessonScheduler.Client = cs

	// Client for creating new CRD definitions
	csExt, err := kubernetesExt.NewForConfig(kubeConfig)
	if err != nil {
		log.Error(err)
		log.Fatalf("Invalid kubeconfig")
	}
	lessonScheduler.ClientExt = csExt

	// Client for creating instances of the network CRD
	clientRest, scheme, err := crd.NewClient(kubeConfig)
	if err != nil {
		log.Error(err)
		log.Fatalf("Invalid kubeconfig")
	}
	// IMPORTANT - for some reason, the client requires a namespace name when we create it here.
	// We are overriding this with the actual namespace name we wish to use when calling any
	// of the client functions, so the fake namespace name provided here doesn't matter.
	// However, if you don't override, we'll have issues since this NS
	// likely won't exist.
	lessonScheduler.ClientCrd = crdclient.CrdClient(clientRest, scheme, "")

	go func() {
		err = lessonScheduler.Start()
		if err != nil {
			log.Fatalf("Problem starting lesson scheduler: %s", err)
		}
	}()

	antidoteSha, err := ioutil.ReadFile(fmt.Sprintf("%s/.git/refs/heads/%s", syringeConfig.LessonDir, syringeConfig.LessonRepoBranch))
	if err != nil {
		log.Error("Encountered problem getting antidote SHA")
		buildInfo["antidoteSha"] = "null"
	} else {
		buildInfo["antidoteSha"] = string(antidoteSha)
	}

	// Start API, and feed it pointer to lesson scheduler so they can talk
	go func() {
		err = api.StartAPI(&lessonScheduler, syringeConfig.GRPCPort, syringeConfig.HTTPPort, buildInfo)
		if err != nil {
			log.Fatalf("Problem starting API: %s", err)
		}
	}()

	// Wait forever
	ch := make(chan struct{})
	<-ch
}
