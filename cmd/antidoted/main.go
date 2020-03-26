package main

import (
	"os"

	crdclient "github.com/nre-learning/antidote-core/pkg/client/clientset/versioned"
	services "github.com/nre-learning/antidote-core/services"
	ot "github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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
		ingestors.ImportCurriculum(adb, config)

		nc, err := nats.Connect(nats.DefaultURL)
		if err != nil {
			log.Fatal(err)
		}
		ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
		if err != nil {
			log.Fatal(err)
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
				NC:            nc,
				NEC:           ec,
				Config:        config,
				Db:            adb,
				BuildInfo:     buildInfo,
				HealthChecker: &scheduler.LessonHealthCheck{},
			}

			// In case we're restarting from a previous instance, we want to make sure we clean up any
			// orphaned k8s namespaces by killing any with our ID
			log.Info("Pruning orphaned namespaces...")
			scheduler.PruneOrphanedNamespaces()

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
				NEC:       ec,
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
				NEC:    ec,
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
