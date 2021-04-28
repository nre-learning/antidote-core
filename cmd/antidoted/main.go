package main

import (
	"os"

	ot "github.com/opentracing/opentracing-go"

	"github.com/nats-io/nats.go"
	api "github.com/nre-learning/antidote-core/api/exp"
	config "github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	ingestors "github.com/nre-learning/antidote-core/db/ingestors"
	scheduler "github.com/nre-learning/antidote-core/scheduler"
	kb "github.com/nre-learning/antidote-core/scheduler/backends/kubernetes"
	"github.com/nre-learning/antidote-core/services"
	stats "github.com/nre-learning/antidote-core/stats"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {

	app := cli.NewApp()
	app.Name = "antidoted"
	app.Version = buildInfo["buildVersion"]
	app.Usage = "The primary back-end service for the Antidote platform"

	var configFile string

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config",
			Usage:       "Configuration file for antidoted",
			Value:       "/etc/antidote/antidote-config.yml",
			Destination: &configFile,
		},
	}

	app.Action = func(c *cli.Context) error {

		log.Infof("antidoted (%s) starting.", buildInfo["buildVersion"])

		config, err := config.LoadConfig(configFile)
		if err != nil {
			log.Fatalf("Failed to read configuration: %v", err)
		}

		tracer, closer := services.InitTracing(config.InstanceID)
		ot.SetGlobalTracer(tracer)
		defer closer.Close()

		buildInfo["curriculumVersion"] = config.CurriculumVersion

		// Initialize DataManager
		adb := db.NewADMInMem()
		err = ingestors.ImportCurriculum(adb, config)
		if err != nil {
			log.Fatal(err)
		}

		nc, err := nats.Connect(config.NATSUrl)
		if err != nil {
			log.Fatal(err)
		}
		defer nc.Close()

		if config.IsServiceEnabled("scheduler") {

			// Initialize backend
			// TODO - loading only kubernetes backend currently. When multiple backends are available, this will be configurable.
			k, err := kb.NewKubernetesBackend(config, adb)

			scheduler := scheduler.AntidoteScheduler{
				Config:    config,
				BuildInfo: buildInfo,
				Backend:   k,
			}

			go func() {
				err = scheduler.Start()
				if err != nil {
					log.Fatalf("Problem starting lesson scheduler: %s", err)
				}
			}()
			log.Info("Scheduler started.")
		}

		if config.IsServiceEnabled("api") {
			apiServer := &api.AntidoteAPI{
				BuildInfo: buildInfo,
				Db:        adb,
				NC:        nc,
				Config:    config,
			}
			go func() {
				err = apiServer.Start()
				if err != nil {
					log.Fatalf("Problem starting API: %s", err)
				}
			}()
			log.Info("API server started.")
		}

		if config.IsServiceEnabled("stats") {
			stats := &stats.AntidoteStats{
				Config: config,
				Db:     adb,
				NC:     nc,
			}
			go func() {
				err = stats.Start()
				if err != nil {
					log.Fatalf("Problem starting Stats: %s", err)
				}
			}()
			log.Info("Stats service started.")
		}

		// Wait forever
		ch := make(chan struct{})
		<-ch

		return nil
	}
	app.Run(os.Args)
}
