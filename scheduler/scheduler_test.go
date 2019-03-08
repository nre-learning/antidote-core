package scheduler

import (
	"os"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	kubernetesExtFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			LessonId: 1,
			Stages: []*pb.LessonStage{
				{
					Id:          0,
					Description: "",
				},
				{
					Id:          1,
					Description: "foobar",
				},
			},
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

	nsName := "1-foobar-ns"
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: nsName,
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

		Client:    testclient.NewSimpleClientset(namespace),
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
		Uuid:      "abcdef",
		Created:   time.Now(),
	}
	lessonScheduler.Requests <- req

	for {
		result := <-lessonScheduler.Results
		log.Info(result)

		if !result.Success && result.Operation == OperationType_CREATE {
			t.Fatal("Received error from scheduler")
		} else if result.Success {
			break
		}
	}

	// TODO(mierdin): Need to create a tester, an

}
