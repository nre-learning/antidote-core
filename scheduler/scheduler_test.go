package scheduler

import (
	"os"
	"sync"
	"testing"
	"time"

	kubernetesExtFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	testclient "k8s.io/client-go/kubernetes/fake"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
)

func TestSchedulerSetup(t *testing.T) {

	os.Setenv("SYRINGE_LESSONS", "foo")
	os.Setenv("SYRINGE_DOMAIN", "bar")
	syringeConfig, err := config.LoadConfigVars()
	if err != nil {
		t.Fatal(err)
	}

	log.Info(syringeConfig)

	var lessonDefs = map[int32]*pb.LessonDef{
		1: &pb.LessonDef{
			LessonId:        1,
			Stages:          []*pb.LessonStage{},
			LessonName:      "Test Lesson",
			IframeResources: []*pb.IframeResource{},
			Devices:         []*pb.Device{},
			Utilities:       []*pb.Utility{},
			Blackboxes:      []*pb.Blackbox{},
			Connections:     []*pb.Connection{},
			Category:        "fundamentals",
			Tier:            "prod",
		},
	}

	// Start lesson scheduler
	lessonScheduler := LessonScheduler{
		// KubeConfig:    kubeConfig,
		Requests:      make(chan *LessonScheduleRequest),
		Results:       make(chan *LessonScheduleResult),
		LessonDefs:    lessonDefs,
		SyringeConfig: syringeConfig,
		GcWhiteList:   make(map[string]*pb.Session),
		GcWhiteListMu: &sync.Mutex{},
		KubeLabs:      make(map[string]*KubeLab),
		KubeLabsMu:    &sync.Mutex{},

		Client:    testclient.NewSimpleClientset(),
		ClientExt: kubernetesExtFake.NewSimpleClientset(),
		// ClientCrd: kubernetesExtFake
	}

	// Start scheduler
	go func() {
		err := lessonScheduler.Start()
		if err != nil {
			t.Fatalf("Problem starting lesson scheduler: %s", err)
		}
	}()

	req := &LessonScheduleRequest{
		LessonDef: lessonDefs[1],
		Operation: OperationType_CREATE,
		Stage:     1,
		Uuid:      "lessonUuid",
		Created:   time.Now(),
	}
	lessonScheduler.Requests <- req

	// TODO(mierdin) How to mock connectivity tests?

	// --------------

	// // Client for creating new CRD definitions
	// csExt, err := kubernetesExt.NewForConfig(kubeConfig)
	// if err != nil {
	// 	log.Error(err)
	// 	log.Fatalf("Invalid kubeconfig")
	// }
	// lessonScheduler.ClientExt = csExt

	// // Client for creating instances of the network CRD
	// clientRest, scheme, err := crd.NewClient(kubeConfig)
	// if err != nil {
	// 	log.Error(err)
	// 	log.Fatalf("Invalid kubeconfig")
	// }
	// lessonScheduler.ClientCrd = crdclient.CrdClient(clientRest, scheme, "")

	// cases := []struct {
	// 	ns string
	// }{
	// 	{
	// 		ns: "test",
	// 	},
	// }

	// api := &KubernetesAPI{
	// 	Suffix: "unit-test",
	// 	Client:  testclient.NewSimpleClientset(),
	// }

	// for _, c := range cases {
	// 	// create the postfixed namespace
	// 	err := api.NewNamespaceWithSuffix(c.ns)
	// 	if err != nil {
	// 		t.Fatal(err.Error())
	// 	}

	// 	_, err = api.Client.CoreV1().Namespaces().Get("test-unit-test", v1.GetOptions{})

	// 	if err != nil {
	// 		t.Fatal(err.Error())
	// 	}
	// }
}
