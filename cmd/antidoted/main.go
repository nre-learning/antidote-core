package main

import (
	"fmt"
	"io/ioutil"
	"sync"

	log "github.com/sirupsen/logrus"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"

	api "github.com/nre-learning/syringe/api/exp"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	crdclient "github.com/nre-learning/syringe/pkg/client/clientset/versioned"
	"github.com/nre-learning/syringe/scheduler"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {

	log.Infof("syringed (%s) starting.", buildInfo["buildVersion"])

	syringeConfig, err := config.LoadConfigVars()
	if err != nil {
		log.Error(err)
		log.Fatalf("Invalid configuration. Please re-run Syringe with appropriate env variables")
	}

	// TODO(mierdin): Now that there's a new package for data management, please create an instance of it here,
	// and pass it equally into both the API server and the scheduler via pointer

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

	// Start lesson scheduler
	lessonScheduler := scheduler.LessonScheduler{
		KubeConfig:    kubeConfig,
		Requests:      make(chan *scheduler.LessonScheduleRequest),
		Results:       make(chan *scheduler.LessonScheduleResult),
		Curriculum:    curriculum,
		SyringeConfig: syringeConfig,
		GcWhiteList:   make(map[string]*pb.Session),
		GcWhiteListMu: &sync.Mutex{},
		KubeLabs:      make(map[string]*scheduler.KubeLab),
		KubeLabsMu:    &sync.Mutex{},
		BuildInfo:     buildInfo,
		HealthChecker: &scheduler.LessonHealthCheck{},
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
	clientCrd, err := crdclient.NewForConfig(kubeConfig)
	if err != nil {
		log.Error(err)
		log.Fatalf("Invalid kubeconfig")
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

	antidoteSha, err := ioutil.ReadFile(fmt.Sprintf("%s/.git/refs/heads/%s", syringeConfig.CurriculumDir, syringeConfig.CurriculumRepoBranch))
	if err != nil {
		log.Error("Encountered problem getting antidote SHA")
		buildInfo["antidoteSha"] = "null"
	} else {
		buildInfo["antidoteSha"] = string(antidoteSha)
	}

	// Start API, and feed it pointer to lesson scheduler so they can talk
	apiServer := &api.SyringeAPIServer{
		LiveLessonState:     make(map[string]*pb.LiveLesson),
		LiveLessonsMu:       &sync.Mutex{},
		VerificationTasks:   make(map[string]*pb.VerificationTask),
		VerificationTasksMu: &sync.Mutex{},
		Scheduler:           &lessonScheduler,
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
