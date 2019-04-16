package api

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/config"
	scheduler "github.com/nre-learning/syringe/scheduler"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

func TestGCWithAPIServerCreatedLessons(t *testing.T) {

	fakeScheduler := createFakeScheduler()
	go func() {
		err := fakeScheduler.Start()
		if err != nil {
			t.Fatalf("Problem starting lesson scheduler: %s", err)
		}
	}()

	s := &SyringeAPIServer{
		LiveLessonState: map[string]*pb.LiveLesson{},
		LiveLessonsMu:   &sync.Mutex{},
		Scheduler:       fakeScheduler,
	}
	go func() {
		err := s.StartAPI(fakeScheduler, nil)
		if err != nil {
			log.Fatalf("Problem starting API: %s", err)
		}
	}()

	numberCreateRequests := 50
	for i := 1; i <= numberCreateRequests; i++ {
		uuid, _ := newUUID()
		s.RequestLiveLesson(nil, &pb.LessonParams{
			LessonId:    1,
			SessionId:   uuid,
			LessonStage: 1,
		})
	}

	time.Sleep(5 * time.Second)

	assert(t, (len(s.LiveLessonState) == numberCreateRequests),
		fmt.Sprintf("livelessonstate has %d members, expected %d", len(s.LiveLessonState), numberCreateRequests))

	////////////////////////////

	// Testing insta-GC
	fakeScheduler.SyringeConfig.LessonTTL = 0

	cleaned, err := fakeScheduler.PurgeOldLessons()
	ok(t, err)

	// assert(t, false, "")
	time.Sleep(5 * time.Second)

	assert(t, (len(cleaned) == numberCreateRequests),
		fmt.Sprintf("cleaned is equal to %d, expected %d", len(cleaned), numberCreateRequests))
	for i := range cleaned {
		// Clean up local kubelab state
		// fakeScheduler.deleteKubelab(cleaned[i])

		// Send result to API server to clean up livelesson state
		fakeScheduler.Results <- &scheduler.LessonScheduleResult{
			Success:   true,
			LessonDef: nil,
			Uuid:      cleaned[i],
			Operation: scheduler.OperationType_DELETE,
		}
	}

	assert(t, (len(s.LiveLessonState) == 0),
		fmt.Sprintf("livelessonstate has %d members, expected %d", len(s.LiveLessonState), 0))

}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// This is a duplicate function from something that's available in teh scheduler test package.
// Since we can't use those files in this package. Is it possible to override this?
func createFakeScheduler() *scheduler.LessonScheduler {
	os.Setenv("SYRINGE_LESSONS", "foo")
	os.Setenv("SYRINGE_DOMAIN", "bar")
	syringeConfig, err := config.LoadConfigVars()
	if err != nil {
		// t.Fatal(err)
		panic(err)
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
			Devices:         []*pb.Endpoint{},
			Utilities: []*pb.Endpoint{
				{
					Name:  "linux1",
					Type:  pb.Endpoint_UTILITY,
					Image: "antidotelabs/utility",
				},
			},
			Blackboxes:  []*pb.Endpoint{},
			Connections: []*pb.Connection{},
			Category:    "fundamentals",
			Tier:        "prod",
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
	lessonScheduler := scheduler.LessonScheduler{
		// KubeConfig:    kubeConfig,
		Requests:      make(chan *scheduler.LessonScheduleRequest),
		Results:       make(chan *scheduler.LessonScheduleResult),
		LessonDefs:    lessonDefs,
		SyringeConfig: syringeConfig,
		GcWhiteList:   make(map[string]*pb.Session),
		GcWhiteListMu: &sync.Mutex{},
		KubeLabs:      make(map[string]*scheduler.KubeLab),
		KubeLabsMu:    &sync.Mutex{},

		DisableGC: true,

		Client:    testclient.NewSimpleClientset(namespace),
		ClientExt: kubernetesExtFake.NewSimpleClientset(),
		ClientCrd: kubernetesCrdFake.NewSimpleClientset(),
	}
	return &lessonScheduler
}
