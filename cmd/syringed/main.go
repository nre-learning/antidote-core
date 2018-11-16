package main

import (
	"fmt"
	"io/ioutil"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	api "github.com/nre-learning/syringe/api/exp"
	config "github.com/nre-learning/syringe/config"
	"github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
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
	}
	go func() {
		err = lessonScheduler.Start()
		if err != nil {
			log.Fatalf("Problem starting lesson scheduler: %s", err)
		}
	}()

	antidoteSha, err := ioutil.ReadFile(fmt.Sprintf("%s/.git/refs/heads/%s", syringeConfig.LessonRepoDir, syringeConfig.LessonRepoBranch))
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
