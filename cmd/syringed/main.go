package main

import (
	"fmt"
	"os"
	"path/filepath"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	log "github.com/Sirupsen/logrus"
	api "github.com/nre-learning/syringe/api/exp"
	config "github.com/nre-learning/syringe/config"
	"github.com/nre-learning/syringe/def"
	"github.com/nre-learning/syringe/scheduler"
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

	// Get lesson definitions
	fileList := []string{}
	log.Debugf("Searching %s for lesson definitions", syringeConfig.LessonsDir)
	err = filepath.Walk(syringeConfig.LessonsDir, func(path string, f os.FileInfo, err error) error {
		syringeFileLocation := fmt.Sprintf("%s/syringe.yaml", path)
		if _, err := os.Stat(syringeFileLocation); err == nil {
			log.Debugf("Found lesson definition at: %s", syringeFileLocation)
			fileList = append(fileList, syringeFileLocation)
		}
		return nil
	})

	lessonDefs, err := def.ImportLessonDefs(syringeConfig, fileList)
	if err != nil {
		log.Warn(err)
	}
	log.Infof("Imported %d lesson definitions.", len(lessonDefs))

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

	// Start API, and feed it pointer to lesson scheduler so they can talk
	go func() {
		err = api.StartAPI(&lessonScheduler, syringeConfig.GRPCPort, syringeConfig.HTTPPort)
		if err != nil {
			log.Fatalf("Problem starting API: %s", err)
		}
	}()

	// Wait forever
	ch := make(chan struct{})
	<-ch
}
