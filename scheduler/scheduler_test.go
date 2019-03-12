package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"

	// Fake clients
	kubernetesCrdFake "github.com/nre-learning/syringe/pkg/client/clientset/versioned/fake"
	kubernetesExtFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Helper functions courtesy of the venerable Ben Johnson
// https://medium.com/@benbjohnson/structuring-tests-in-go-46ddee7a25c

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

func TestSchedulerSetup(t *testing.T) {

	os.Setenv("SYRINGE_LESSONS", "foo")
	os.Setenv("SYRINGE_DOMAIN", "bar")
	syringeConfig, err := config.LoadConfigVars()
	if err != nil {
		t.Fatal(err)
	}

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
			Devices: []*pb.Endpoint{
				{
					Name:  "vqfx1",
					Type:  pb.Endpoint_DEVICE,
					Image: "antidotelabs/vqfx",
				},
				{
					Name:  "vqfx2",
					Type:  pb.Endpoint_DEVICE,
					Image: "antidotelabs/vqfx",
				},
				{
					Name:  "vqfx3",
					Type:  pb.Endpoint_DEVICE,
					Image: "antidotelabs/vqfx",
				},
			},
			Utilities: []*pb.Endpoint{
				{
					Name:  "linux1",
					Type:  pb.Endpoint_UTILITY,
					Image: "antidotelabs/utility",
				},
			},
			Blackboxes: []*pb.Endpoint{},
			Connections: []*pb.Connection{
				{
					A: "vqfx1",
					B: "vqfx2",
				},
				{
					A: "vqfx2",
					B: "vqfx3",
				},
				{
					A: "vqfx3",
					B: "vqfx1",
				},
			},
			Category: "fundamentals",
			Tier:     "prod",
		},
	}

	nsName := "1-foobar-ns"
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			// Namespace: nsName,
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
		ClientCrd: kubernetesCrdFake.NewSimpleClientset(),
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

	// TODO(mierdin): Need to create a fake health check tester

}
