package scheduler

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	models "github.com/nre-learning/antidote-core/db/models"
	ot "github.com/opentracing/opentracing-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNamespaces is responsible for ensuring kubernetes namespaces are created as expected, with expected
// properties set based on Syringe-specific inputs.
func TestNamespaces(t *testing.T) {

	span := ot.StartSpan("test_db")
	defer span.Finish()

	nsName := "100-foobar-ns"
	schedulerSvc := createFakeScheduler()
	anHourAgo := time.Now().Add(time.Duration(-1) * time.Hour)
	schedulerSvc.Db.CreateLiveSession(span.Context(), &models.LiveSession{
		ID: "abcdef",
	})

	// Test basic namespace creation
	t.Run("A=1", func(t *testing.T) {
		namespaces := []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"lessonId":        "100",
						"antidoteManaged": "yes",
						"liveSession":     "abcdef",
						"liveLesson":      "123456",
						"name":            nsName,
						"antidoteId":      schedulerSvc.Config.InstanceID,
						"lastAccessed":    strconv.Itoa(int(anHourAgo.Unix())),
						"created":         strconv.Itoa(int(anHourAgo.Unix())),
					},
				},
			},
		}

		for n := range namespaces {
			ns, err := schedulerSvc.Client.CoreV1().Namespaces().Create(namespaces[n])

			// Assert namespace exists without error
			ok(t, err)
			assert(t, (ns != nil), "")
		}
	})

	// Test that namespaces are GC'd as expected.
	t.Run("A=1", func(t *testing.T) {
		cleaned, err := schedulerSvc.PurgeOldLessons(span.Context())
		ok(t, err)
		assert(t, (len(cleaned) == 1), fmt.Sprintf("%d", len(cleaned)))
		assert(t, (cleaned[0] == "123456"), cleaned[0])
	})

}

func TestSessionPersistence(t *testing.T) {
	span := ot.StartSpan("test_db")
	defer span.Finish()

	nsName1 := "100-foobar-ns"
	nsName2 := "200-foobar-ns"
	schedulerSvc := createFakeScheduler()
	anHourAgo := time.Now().Add(time.Duration(-1) * time.Hour)

	schedulerSvc.Db.CreateLiveSession(span.Context(), &models.LiveSession{
		ID:         "abcdef",
		Persistent: false,
	})

	schedulerSvc.Db.CreateLiveSession(span.Context(), &models.LiveSession{
		ID:         "uvwxyz",
		Persistent: true,
	})

	// Test basic namespace creation
	t.Run("A=1", func(t *testing.T) {
		namespaces := []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName1,
					Labels: map[string]string{
						"lessonId":        "100",
						"antidoteManaged": "yes",
						"liveSession":     "abcdef",
						"liveLesson":      "123456",
						"name":            nsName1,
						"antidoteId":      schedulerSvc.Config.InstanceID,
						"lastAccessed":    strconv.Itoa(int(anHourAgo.Unix())),
						"created":         strconv.Itoa(int(anHourAgo.Unix())),
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName2,
					Labels: map[string]string{
						"lessonId":        "200",
						"antidoteManaged": "yes",
						"liveSession":     "uvwxyz",
						"liveLesson":      "987654",
						"name":            nsName2,
						"antidoteId":      schedulerSvc.Config.InstanceID,
						"lastAccessed":    strconv.Itoa(int(anHourAgo.Unix())),
						"created":         strconv.Itoa(int(anHourAgo.Unix())),
					},
				},
			},
		}

		for n := range namespaces {
			ns, err := schedulerSvc.Client.CoreV1().Namespaces().Create(namespaces[n])

			// Assert namespace exists without error
			ok(t, err)
			assert(t, (ns != nil), "")
		}
	})

	// Test that namespaces are GC'd as expected.
	t.Run("A=1", func(t *testing.T) {
		cleaned, err := schedulerSvc.PurgeOldLessons(span.Context())
		ok(t, err)
		assert(t, (len(cleaned) == 1), fmt.Sprintf("%d", len(cleaned)))
		assert(t, (cleaned[0] == "123456"), cleaned[0])
	})
}
