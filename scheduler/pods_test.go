package scheduler

import (
	"fmt"
	"testing"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
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
	syringeConfig := &config.SyringeConfig{
		LessonsDir: "/antidote",
		Domain:     "localhost",
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: nsName,
		},
	}
	lessonScheduler := LessonScheduler{
		SyringeConfig: syringeConfig,
		Client:        testclient.NewSimpleClientset(namespace),
		ClientExt:     kubernetesExtFake.NewSimpleClientset(),
	}
	uuid := "1-abcdef"
	// END SETUP

	// Test normal pod creation
	t.Run("A=1", func(t *testing.T) {

		pod, err := lessonScheduler.createPod(
			&pb.Endpoint{
				Name:  "linux1",
				Type:  pb.Endpoint_UTILITY,
				Image: "antidotelabs/utility",
			},
			[]string{"1", "2", "3"},
			&LessonScheduleRequest{
				Uuid: uuid,
				LessonDef: &pb.LessonDef{
					LessonId: 1,
				},
			},
		)

		// Assert pod exists without error
		ok(t, err)
		assert(t, (pod != nil), "")

		// Assert created namespace is correct
		equals(t, pod.Namespace, fmt.Sprintf("%s-ns", uuid))

		// Assert expected networks exist properly

		// t.Log(pod)
	})

	// Test bad pod creation
	t.Run("A=1", func(t *testing.T) {

		pod, err := lessonScheduler.createPod(
			&pb.Endpoint{
				Name: "linux1",

				// Lots of stuff happens if the type of the endpoint is not known.
				// Such as failing to assign ports. We want to test for this.
				Type:  pb.Endpoint_UNKNOWN,
				Image: "antidotelabs/utility",
			},
			[]string{"1", "2", "3"},
			&LessonScheduleRequest{
				Uuid: uuid,
				LessonDef: &pb.LessonDef{
					LessonId: 1,
				},
			},
		)

		// Assert pod did not get created
		assert(t, (pod == nil), "")
		assert(t, (err != nil), "")
	})

}
