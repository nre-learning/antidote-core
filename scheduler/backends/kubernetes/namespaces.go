package kubernetes

import (
	"fmt"
	"strconv"
	"sync"
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

func (k *KubernetesBackend) boopNamespace(sc ot.SpanContext, nsName string) error {

	span := ot.StartSpan("scheduler_boop_ns", ot.ChildOf(sc))
	defer span.Finish()

	ns, err := k.Client.CoreV1().Namespaces().Get(nsName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ns.ObjectMeta.Labels["lastAccessed"] = strconv.Itoa(int(time.Now().Unix()))

	_, err = k.Client.CoreV1().Namespaces().Update(ns)
	if err != nil {
		return err
	}

	return nil
}

// PruneOrphanedNamespaces seeks out all antidote-managed namespaces, and deletes them.
// This will effectively reset the cluster to a state with all of the remaining infrastructure
// in place, but no running lessons. Antidote doesn't manage itself, or any other Antidote services.
func (k *KubernetesBackend) PruneOrphanedNamespaces() error {

	span := ot.StartSpan("scheduler_prune_orphaned_ns")
	defer span.Finish()

	nameSpaces, err := k.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll nuke way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", k.Config.InstanceID),
	})
	if err != nil {
		return err
	}

	// No need to nuke if no namespaces exist with our ID
	if len(nameSpaces.Items) == 0 {
		span.LogFields(log.Int("pruned_orphans", 0))
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(len(nameSpaces.Items))
	for n := range nameSpaces.Items {

		nsName := nameSpaces.Items[n].ObjectMeta.Name
		go func() {
			defer wg.Done()
			s.deleteNamespace(span.Context(), nsName)
		}()
	}
	wg.Wait()

	span.LogFields(log.Int("pruned_orphans", len(nameSpaces.Items)))
	return nil
}

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
func generateNamespaceName(antidoteID, liveLessonID string) string {
	return fmt.Sprintf("%s-%s", antidoteID, liveLessonID)
}
