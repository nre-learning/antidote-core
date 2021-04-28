package kubernetes

import (
	"crypto/rand"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/nre-learning/antidote-core/config"
	db "github.com/nre-learning/antidote-core/db"
	ingestors "github.com/nre-learning/antidote-core/db/ingestors"

	// Fake clients
	kubernetesCrdFake "github.com/nre-learning/antidote-core/pkg/client/clientset/versioned/fake"
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

type fakeHealthChecker struct{}

func (lhc *fakeHealthChecker) sshTest(host string, port int) bool { return true }
func (lhc *fakeHealthChecker) tcpTest(host string, port int) bool { return true }

func createFakeKubernetesBackend() *KubernetesBackend {
	cfg, err := config.LoadConfig("../hack/mocks/mock-config-1.yml")
	if err != nil {
		// t.Fatal(err)
		panic(err)
	}

	// Initialize DataManager
	adb := db.NewADMInMem()
	err = ingestors.ImportCurriculum(adb, cfg)
	if err != nil {
		panic(err)
	}

	nsName := "1-foobar-ns"
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			// Namespace: nsName,
		},
	}

	// Start lesson scheduler
	kb := KubernetesBackend{
		// KubeConfig:    kubeConfig,
		Config:    cfg,
		Client:    testclient.NewSimpleClientset(namespace),
		ClientExt: kubernetesExtFake.NewSimpleClientset(),
		ClientCrd: kubernetesCrdFake.NewSimpleClientset(),
		// NEC:       ec,
		Db: adb,
	}

	return &kb
}

// TODO - this is from pre-backend
// func TestSchedulerSetup(t *testing.T) {

// 	lessonScheduler := createFakeScheduler()

// 	// Start scheduler
// 	go func() {
// 		err := lessonScheduler.Start()
// 		if err != nil {
// 			t.Fatalf("Problem starting lesson scheduler: %s", err)
// 		}
// 	}()

// 	// TODO(mierdin): The previous edition for this test (pre rewrite) sent LSRs into the scheduler channel, and then
// 	// made assertions about the kubelabs that were created and what state they were in (k8s objects)
// 	// We could probably do the same thing post-rewrite but obviously kubelab is gone so likely what we'd have to do is
// 	// call out to kube directly in these tests in order to make the same assertions.
// 	// The final thing these tests did was perform garbage collection, and then make additional assertions accordingly.

// }

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
