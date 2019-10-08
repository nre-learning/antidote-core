package main

import (

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	config "github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
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

	// curriculum, err := api.ImportCurriculum(syringeConfig)
	// if err != nil {
	// 	log.Warn(err)
	// }

	apiServer := &MockAPIServer{}
	err = apiServer.StartAPI(syringeConfig)
	if err != nil {
		log.Fatalf("Problem starting API: %s", err)
	}
}
