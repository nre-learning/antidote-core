package kubernetes

import (
	"fmt"
	"strconv"
	"time"

	ot "github.com/opentracing/opentracing-go"

	"github.com/nre-learning/antidote-core/services"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	// Kubernetes types
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubernetesBackend) deleteNamespace(sc ot.SpanContext, name string) error {

	span := ot.StartSpan("scheduler_delete_ns", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("nsName", name)

	err := k.Client.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	// Wait for the namespace to be deleted
	deleteTimeoutSecs := 120
	for i := 0; i < deleteTimeoutSecs/5; i++ {
		time.Sleep(5 * time.Second)

		_, err := k.Client.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
		if err == nil {
			continue
		} else if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	err = fmt.Errorf("Timed out trying to delete namespace %s", name)
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return err
}

func (k *KubernetesBackend) createNamespace(sc ot.SpanContext, req services.LessonScheduleRequest) (*corev1.Namespace, error) {
	span := ot.StartSpan("scheduler_create_namespace", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, req.LiveLessonID)
	span.LogFields(log.String("nsName", nsName))

	ll, err := k.Db.GetLiveLesson(span.Context(), req.LiveLessonID)
	if err != nil {
		return nil, err
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"name":            nsName, // IMPORTANT - used by networkpolicy to restrict traffic
				"liveLesson":      fmt.Sprintf("%s", req.LiveLessonID),
				"liveSession":     fmt.Sprintf("%s", req.LiveSessionID),
				"lessonSlug":      fmt.Sprintf("%s", ll.LessonSlug),
				"antidoteManaged": "yes",
				"antidoteId":      k.Config.InstanceID,
				"lastAccessed":    strconv.Itoa(int(ll.CreatedTime.Unix())),
				"created":         strconv.Itoa(int(ll.CreatedTime.Unix())),
			},
		},
	}

	result, err := k.Client.CoreV1().Namespaces().Create(namespace)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}
	return result, err
}

// generateNamespaceName is a helper function for determining the name of our kubernetes
// namespaces, so we don't have to do this all over the codebase and maybe get it wrong.
// Note that the nsName is used EVERYWHERE, and what's in it is pretty important, so change
// this formatting with CAUTION. For instance, the antidoteId is how we disambiguate between
// instances for HEPS domains. **MAKE SURE** that this formatting matches the creation of
// nsName in the API server right before the initializeLiveEndpoints function.
// TODO(mierdin): Make this less dependent on the honor system.
// TODO(mierdin): This should be moved out to scheduler utils or something - so that all backends can get the same formatting.
// You may want to create a struct type that holds the antidoteID and the livelesson ID, and generate a string from those components,
// so you can mandate that this type is used where needed.
func generateNamespaceName(antidoteID, liveLessonID string) string {
	return fmt.Sprintf("%s-%s", antidoteID, liveLessonID)
}
