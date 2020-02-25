package main

import (
	crdclient "github.com/nre-learning/syringe/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	api "github.com/nre-learning/syringe/api/exp"
	config "github.com/nre-learning/syringe/config"
	db "github.com/nre-learning/syringe/db"
	"github.com/nre-learning/syringe/scheduler"
	"github.com/nre-learning/syringe/services"
	stats "github.com/nre-learning/syringe/stats"
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

	// Because channels are synchronous, and one-to-one, we need one channel for each unique
	// communication path. Hopefully we can revisit this soon and have more of a true one-to-many solution
	// that doesn't require an external server like NATS, but for now this works.
	eb := services.NewEventBus()
	// This channel allows the API service to send requests to the scheduler
	apiToScheduler := make(chan services.LessonScheduleRequest)
	eb.Subscribe("lesson.requested", apiToScheduler)
	// This channel allows the scheduler service to send updates to the stats service
	schedulerToStats := make(chan services.LessonScheduleRequest)
	eb.Subscribe("lesson.started", schedulerToStats)

	if config.IsServiceEnabled("scheduler") {
		var kubeConfig *rest.Config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
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
			KubeClient:    cs,
			KubeClientExt: csExt,
			KubeClientCrd: clientCrd,
			Requests:      apiToScheduler,
			Results:       schedulerToStats,
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
			Config:    config,
			Requests:  apiToScheduler,
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
			Reqs:   schedulerToStats,
			Config: config,
			Db:     adb,
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
