package scheduler

import (
	"fmt"
	"testing"

	config "github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
	services "github.com/nre-learning/antidote-core/services"
	corev1 "k8s.io/api/core/v1"
	kubernetesExtFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// TestPods is responsible for ensuring kubernetes pods are created as expected, with expected
// properties set based on Syringe-specific inputs.
func TestPods(t *testing.T) {

	// SETUP
	nsName := "1-foobar-ns"
	cfg := config.AntidoteConfig{
		CurriculumDir: "/antidote",
		Domain:        "localhost",
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: nsName,
		},
	}
	lessonScheduler := AntidoteScheduler{
		Config:    cfg,
		Client:    testclient.NewSimpleClientset(namespace),
		ClientExt: kubernetesExtFake.NewSimpleClientset(),
	}
	uuid := "1-abcdef"
	// END SETUP

	// Test normal pod creation
	t.Run("A=1", func(t *testing.T) {

		pod, err := lessonScheduler.createPod(
			&models.LiveEndpoint{
				Name:  "linux1",
				Image: "antidotelabs/utility",
				Presentations: []*models.LivePresentation{
					{Name: "cli", Type: "ssh", Port: 22},
				},
			},
			[]string{"1", "2", "3"},
			services.LessonScheduleRequest{
				LiveLessonID: "asdf",
			},
		)

		// Assert pod exists without error
		ok(t, err)
		assert(t, (pod != nil), "")

		// Assert created namespace is correct
		equals(t, pod.Namespace, fmt.Sprintf("%s-ns", uuid))

		// TODO(mierdin): Assert expected networks exist properly

	})

}
