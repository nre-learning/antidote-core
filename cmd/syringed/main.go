package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	log "github.com/Sirupsen/logrus"
	api "github.com/nre-learning/syringe/api/exp"
	"github.com/nre-learning/syringe/def"
	"github.com/nre-learning/syringe/scheduler"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {

	// Get configuration parameters from env
	searchDir := os.Getenv("SYRINGE_LESSONS")
	if searchDir == "" {
		log.Fatalf("Please re-run syringed with SYRINGE_LESSONS environment variable set")
	}

	grpcPort, _ := strconv.Atoi(os.Getenv("SYRINGE_GRPC_PORT"))
	if grpcPort == 0 {
		grpcPort = 50099
	}
	httpPort, _ := strconv.Atoi(os.Getenv("SYRINGE_HTTP_PORT"))
	if httpPort == 0 {
		httpPort = 8086
	}

	// get config
	var useKubeConfig bool
	kcStr := os.Getenv("SYRINGE_KUBECONFIG")
	if kcStr == "yes" {
		useKubeConfig = true
	} else {
		useKubeConfig = false
	}
	config := getConfig(useKubeConfig)

	// Get lab definitions
	fileList := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		syringeFileLocation := fmt.Sprintf("%s/syringe.yaml", path)
		if _, err := os.Stat(syringeFileLocation); err == nil {
			fileList = append(fileList, syringeFileLocation)
		}
		return nil
	})
	labDefs, err := def.ImportLabDefs(fileList)
	if err != nil {
		log.Warn(err)
	}
	log.Infof("Imported %d lab definitions.", len(labDefs))

	// Start lab scheduler
	labScheduler := scheduler.LabScheduler{
		Config:   config,
		Requests: make(chan *scheduler.LabScheduleRequest),
		Results:  make(chan *scheduler.LabScheduleResult),
		LabDefs:  labDefs,
	}
	go func() {
		err = labScheduler.Start()
		if err != nil {
			log.Fatalf("Problem starting lab scheduler: %s", err)
		}
	}()

	// Start API, and feed it pointer to lab scheduler so they can talk
	go func() {
		err = api.StartAPI(&labScheduler, grpcPort, httpPort)
		if err != nil {
			log.Fatalf("Problem starting API: %s", err)
		}
	}()

	// Wait forever
	ch := make(chan struct{})
	<-ch
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getConfig(useKubeConfig bool) *rest.Config {
	if useKubeConfig {
		var kubeconfig string
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		// flag.Parse()
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
		return config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
		return config
	}
}
