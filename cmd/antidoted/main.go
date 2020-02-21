package main

import (
	log "github.com/sirupsen/logrus"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"

	api "github.com/nre-learning/syringe/api/exp"
	config "github.com/nre-learning/syringe/config"
	db "github.com/nre-learning/syringe/db"
	crdclient "github.com/nre-learning/syringe/pkg/client/clientset/versioned"
	"github.com/nre-learning/syringe/scheduler"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {

	log.Infof("antidoted (%s) starting.", buildInfo["buildVersion"])

	syringeConfig, err := config.LoadConfigVars()
	if err != nil {
		log.Fatalf("Invalid configuration. Please re-run Antidote with appropriate env variables - %v", err)
	}

	// TODO(mierdin): This provides the loaded version of the curriculum via syringeinfo, primarily
	// for the PTR banner on the front-end. Should rename to something that makes sense
	buildInfo["antidoteSha"] = syringeConfig.CurriculumVersion

	// Initialize DataManager
	adb := db.NewADMInMem()

	var kubeConfig *rest.Config
	if !syringeConfig.DisableScheduler {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		kubeConfig = &rest.Config{}
	}

	curriculum, err := api.ImportCurriculum(syringeConfig)
	if err != nil {
		log.Warn(err)
	}

	// Build comms channels
	req := make(chan *scheduler.LessonScheduleRequest)
	res := make(chan *scheduler.LessonScheduleResult)

	// Start lesson scheduler
	lessonScheduler := scheduler.LessonScheduler{
		KubeConfig:    kubeConfig,
		Requests:      req,
		Results:       res,
		Curriculum:    curriculum,
		SyringeConfig: syringeConfig,
		// GcWhiteList:   make(map[string]*pb.Session),
		// GcWhiteListMu: &sync.Mutex{},
		// KubeLabs:      make(map[string]*scheduler.KubeLab),
		// KubeLabsMu:    &sync.Mutex{},
		Db:            adb,
		BuildInfo:     buildInfo,
		HealthChecker: &scheduler.LessonHealthCheck{},
	}

	// CREATION OF CLIENTS
	//
	// Client for working with standard kubernetes resources
	cs, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Unable to create new kubernetes client - %v", err)
	}
	lessonScheduler.Client = cs

	// Client for creating new CRD definitions
	csExt, err := kubernetesExt.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Unable to create new kubernetes ext client - %v", err)
	}
	lessonScheduler.ClientExt = csExt

	// Client for creating instances of the network CRD
	clientCrd, err := crdclient.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Unable to create new kubernetes crd client - %v", err)
	}
	lessonScheduler.ClientCrd = clientCrd

	if !syringeConfig.DisableScheduler {
		go func() {
			err = lessonScheduler.Start()
			if err != nil {
				log.Fatalf("Problem starting lesson scheduler: %s", err)
			}
		}()
	} else {
		log.Info("Skipping scheduler start due to configuration")
	}

	// Start API, and feed it pointer to lesson scheduler so they can talk
	apiServer := &api.SyringeAPIServer{
		// LiveLessonState:     make(map[string]*pb.LiveLesson),
		// LiveLessonsMu:       &sync.Mutex{},
		// VerificationTasks:   make(map[string]*pb.VerificationTask),
		// VerificationTasksMu: &sync.Mutex{},
		// Scheduler:           &lessonScheduler,

		Db:            adb,
		SyringeConfig: syringeConfig,
		Requests:      req,
		Results:       res,
	}
	go func() {
		err = apiServer.StartAPI(&lessonScheduler, buildInfo)
		if err != nil {
			log.Fatalf("Problem starting API: %s", err)
		}
	}()

	// Wait forever
	ch := make(chan struct{})
	<-ch
}
