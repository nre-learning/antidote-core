package scheduler

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

func (s *AntidoteScheduler) boopNamespace(sc ot.SpanContext, nsName string) error {

	span := ot.StartSpan("scheduler_boop_ns", ot.ChildOf(sc))
	defer span.Finish()

	ns, err := s.Client.CoreV1().Namespaces().Get(nsName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ns.ObjectMeta.Labels["lastAccessed"] = strconv.Itoa(int(time.Now().Unix()))

	_, err = s.Client.CoreV1().Namespaces().Update(ns)
	if err != nil {
		return err
	}

	return nil
}

// PruneOrphanedNamespaces seeks out all antidote-managed namespaces, and deletes them.
// This will effectively reset the cluster to a state with all of the remaining infrastructure
// in place, but no running lessons. Antidote doesn't manage itself, or any other Antidote services.
func (s *AntidoteScheduler) PruneOrphanedNamespaces() error {

	span := ot.StartSpan("scheduler_prune_orphaned_ns")
	defer span.Finish()

	nameSpaces, err := s.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll nuke way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", s.Config.InstanceID),
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

func (s *AntidoteScheduler) deleteNamespace(sc ot.SpanContext, name string) error {

	span := ot.StartSpan("scheduler_boop_ns", ot.ChildOf(sc))
	defer span.Finish()

	err := s.Client.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	// Wait for the namespace to be deleted
	deleteTimeoutSecs := 120
	for i := 0; i < deleteTimeoutSecs/5; i++ {
		time.Sleep(5 * time.Second)

		_, err := s.Client.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
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

func (s *AntidoteScheduler) createNamespace(sc ot.SpanContext, req services.LessonScheduleRequest) (*corev1.Namespace, error) {
	span := ot.StartSpan("scheduler_create_namespace", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)
	span.LogFields(log.String("nsName", nsName))

	ll, err := s.Db.GetLiveLesson(span.Context(), req.LiveLessonID)
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
				"antidoteId":      s.Config.InstanceID,
				"lastAccessed":    strconv.Itoa(int(ll.CreatedTime.Unix())),
				"created":         strconv.Itoa(int(ll.CreatedTime.Unix())),
			},
		},
	}

	result, err := s.Client.CoreV1().Namespaces().Create(namespace)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}
	return result, err
}

// PurgeOldLessons identifies any kubernetes namespaces that are operating with our antidoteId,
// and among those, deletes the ones that have a lastAccessed timestamp that exceeds our configured
// TTL. This function is meant to be run in a loop within a goroutine, at a configured interval. Returns
// a slice of livelesson IDs to be deleted by the caller (not handled by this function)
func (s *AntidoteScheduler) PurgeOldLessons(sc ot.SpanContext) ([]string, error) {
	span := ot.StartSpan("scheduler_purgeoldlessons", ot.ChildOf(sc))
	defer span.Finish()

	nameSpaces, err := s.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll delete way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", s.Config.InstanceID),
	})
	if err != nil {
		return nil, err
	}

	// No need to GC if no matching namespaces exist
	if len(nameSpaces.Items) == 0 {
		span.LogFields(log.Int("gc_namespaces", 0))
		return []string{}, nil
	}

	liveLessonsToDelete := []string{}
	oldNameSpaces := []string{}
	for n := range nameSpaces.Items {

		// lastAccessed =
		i, err := strconv.ParseInt(nameSpaces.Items[n].ObjectMeta.Labels["lastAccessed"], 10, 64)
		if err != nil {
			return []string{}, err
		}
		lastAccessed := time.Unix(i, 0)

		if time.Since(lastAccessed) < time.Duration(s.Config.LiveLessonTTL)*time.Minute {
			continue
		}

		lsID := nameSpaces.Items[n].ObjectMeta.Labels["liveSession"]
		ls, err := s.Db.GetLiveSession(span.Context(), lsID)
		if err != nil {
			return []string{}, err
		}
		if ls.Persistent {
			span.LogEvent("Skipping GC, session marked persistent")
			continue
		}

		liveLessonsToDelete = append(liveLessonsToDelete, nameSpaces.Items[n].ObjectMeta.Labels["liveLesson"])
		oldNameSpaces = append(oldNameSpaces, nameSpaces.Items[n].ObjectMeta.Name)
	}

	// No need to GC if no old namespaces exist
	if len(oldNameSpaces) == 0 {
		span.LogFields(log.Int("gc_namespaces", 0))
		return []string{}, nil
	}

	var wg sync.WaitGroup
	wg.Add(len(oldNameSpaces))
	for n := range oldNameSpaces {
		go func(ns string) {
			defer wg.Done()
			s.deleteNamespace(span.Context(), ns)
		}(oldNameSpaces[n])
	}
	wg.Wait()
	span.LogFields(log.Int("gc_namespaces", len(oldNameSpaces)))

	return liveLessonsToDelete, nil

}

// generateNamespaceName is a helper function for determining the name of our kubernetes
// namespaces, so we don't have to do this all over the codebase and maybe get it wrong.
func generateNamespaceName(antidoteId, liveLessonID string) string {
	return fmt.Sprintf("%s-%s", antidoteId, liveLessonID)
}
