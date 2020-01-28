package scheduler

import (
	"strconv"
	"testing"
	"time"

	config "github.com/nre-learning/syringe/config"
	corev1 "k8s.io/api/core/v1"
	kubernetesExtFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// TestNamespaces is responsible for ensuring kubernetes namespaces are created as expected, with expected
// properties set based on Syringe-specific inputs.
func TestNamespaces(t *testing.T) {

	// SETUP
	nsName := "100-foobar-ns"
	syringeConfig := &config.SyringeConfig{
		CurriculumDir: "/antidote",
		SyringeID:     "syringe-testing",
		Domain:        "localhost",
		Tier:          "prod",
	}
	lessonScheduler := LessonScheduler{
		SyringeConfig: syringeConfig,
		Client:        testclient.NewSimpleClientset(),
		ClientExt:     kubernetesExtFake.NewSimpleClientset(),
	}
	// END SETUP

	anHourAgo := time.Now().Add(time.Duration(-1) * time.Hour)

	// Test basic namespace creation
	t.Run("A=1", func(t *testing.T) {
		namespaces := []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"lessonId":       "100",
						"syringeManaged": "yes",
						"name":           nsName,
						"syringeId":      lessonScheduler.SyringeConfig.SyringeID,
						"lastAccessed":   strconv.Itoa(int(anHourAgo.Unix())),
						"created":        strconv.Itoa(int(anHourAgo.Unix())),
					},
				},
			},
		}

		for n := range namespaces {
			ns, err := lessonScheduler.Client.CoreV1().Namespaces().Create(namespaces[n])

			// Assert namespace exists without error
			ok(t, err)
			assert(t, (ns != nil), "")
		}
	})

	// Test that namespaces are GC'd as expected.
	t.Run("A=1", func(t *testing.T) {
		cleaned, err := lessonScheduler.PurgeOldLessons()
		ok(t, err)
		assert(t, (len(cleaned) == 1), "")
		assert(t, (cleaned[0] == "100-foobar-ns"), "")
	})

}
