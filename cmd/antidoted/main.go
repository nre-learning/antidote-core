package main

import (
	crdclient "github.com/nre-learning/antidote-core/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"

	nats "github.com/nats-io/nats.go"
	api "github.com/nre-learning/antidote-core/api/exp"
	config "github.com/nre-learning/antidote-core/config"
	db "github.com/nre-learning/antidote-core/db"
	ingestors "github.com/nre-learning/antidote-core/db/ingestors"
	"github.com/nre-learning/antidote-core/scheduler"
	stats "github.com/nre-learning/antidote-core/stats"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {

	log.Infof("antidoted (%s) starting.", buildInfo["buildVersion"])

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Invalid configuration - %v", err)
	}

	// TODO(mierdin): This provides the loaded version of the curriculum via syringeinfo, primarily
	// for the PTR banner on the front-end. Should rename to something that makes sense
	buildInfo["antidoteSha"] = config.CurriculumVersion

	// Initialize DataManager
	adb := db.NewADMInMem()
	ingestors.ImportCurriculum(adb, config)

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		panic(err)
	}
	defer ec.Close()

	if config.IsServiceEnabled("scheduler") {

		// OUT OF CLUSTER CONFIG FOR TESTING
		kubeConfig, err := clientcmd.BuildConfigFromFlags("", "/home/mierdin/.kube/selfmedicateconfig")
		if err != nil {
			panic(err.Error())
		}

		// var kubeConfig *rest.Config
		// kubeConfig, err = rest.InClusterConfig()
		// if err != nil {
		// 	log.Fatal(err)
		// }

		cs, err := kubernetes.NewForConfig(kubeConfig) // Client for working with standard kubernetes resources
		if err != nil {
			log.Fatalf("Unable to create new kubernetes client - %v", err)
		}
		csExt, err := kubernetesExt.NewForConfig(kubeConfig) // Client for creating new CRD definitions
		if err != nil {
			log.Fatalf("Unable to create new kubernetes ext client - %v", err)
		}
		clientCrd, err := crdclient.NewForConfig(kubeConfig) // Client for creating instances of the network CRD
		if err != nil {
			log.Fatalf("Unable to create new kubernetes crd client - %v", err)
		}
		// Start scheduler
		scheduler := scheduler.AntidoteScheduler{
			KubeConfig:    kubeConfig,
			Client:        cs,
			ClientExt:     csExt,
			ClientCrd:     clientCrd,
			NEC:           ec,
			Config:        config,
			Db:            adb,
			BuildInfo:     buildInfo,
			HealthChecker: &scheduler.LessonHealthCheck{},
		}
		go func() {
			err = scheduler.Start()
			if err != nil {
				log.Fatalf("Problem starting lesson scheduler: %s", err)
			}
		}()
	}

	if config.IsServiceEnabled("api") {
		apiServer := &api.AntidoteAPI{
			BuildInfo: buildInfo,
			Db:        adb,
			NEC:       ec,
			Config:    config,
		}
		go func() {
			err = apiServer.Start()
			if err != nil {
				log.Fatalf("Problem starting API: %s", err)
			}
		}()
	}

	if config.IsServiceEnabled("stats") {
		stats := &stats.AntidoteStats{
			Config: config,
			Db:     adb,
			NEC:    ec,
		}
		go func() {
			err = stats.Start()
			if err != nil {
				log.Fatalf("Problem starting Stats: %s", err)
			}
		}()
	}

	// Wait forever
	ch := make(chan struct{})
	<-ch
}
