package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/nre-learning/antidote-core/services"
	log "github.com/sirupsen/logrus"

	// Kubernetes types
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *AntidoteScheduler) boopNamespace(nsName string) error {

	log.Debugf("Booping %s", nsName)

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

// cleanOrphanedNamespaces seeks out all antidote-managed namespaces, and deletes them.
// This will effectively reset the cluster to a state with all of the remaining infrastructure
// in place, but no running lessons. Antidote doesn't manage itself, or any other Antidote services.
func (s *AntidoteScheduler) cleanOrphanedNamespaces() error {

	nameSpaces, err := s.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll nuke way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", s.Config.InstanceID),
	})
	if err != nil {
		return err
	}

	// No need to nuke if no namespaces exist with our ID
	if len(nameSpaces.Items) == 0 {
		log.Info("No namespaces with our antidoteId found. Starting normally.")
		return nil
	}

	log.Warnf("Nuking all namespaces with an antidoteId of %s", s.Config.InstanceID)
	var wg sync.WaitGroup
	wg.Add(len(nameSpaces.Items))
	for n := range nameSpaces.Items {

		nsName := nameSpaces.Items[n].ObjectMeta.Name
		go func() {
			defer wg.Done()
			s.deleteNamespace(nsName)
		}()
	}
	wg.Wait()
	log.Info("Nuke complete. It was the only way to be sure...")
	return nil
}

func (s *AntidoteScheduler) deleteNamespace(name string) error {

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
			log.Debugf("Waiting for namespace %s to delete...", name)
			continue
		} else if apierrors.IsNotFound(err) {
			log.Infof("Deleted namespace %s", name)
			return nil
		} else {
			return err
		}
	}

	errorMsg := fmt.Sprintf("Timed out trying to delete namespace %s", name)
	log.Error(errorMsg)
	return errors.New(errorMsg)
}

func (s *AntidoteScheduler) createNamespace(req services.LessonScheduleRequest) (*corev1.Namespace, error) {

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	log.Infof("Creating namespace: %s", nsName)

	ll, err := s.Db.GetLiveLesson(req.LiveLessonID)
	if err != nil {
		return nil, err
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"liveLesson":      fmt.Sprintf("%s", req.LiveLessonID),
				"liveSession":     fmt.Sprintf("%s", req.LiveSessionID),
				"lessonSlug":      fmt.Sprintf("%s", ll.LessonSlug),
				"antidoteManaged": "yes",
				"antidoteId":      s.Config.InstanceID,
				"lastAccessed":    strconv.Itoa(int(req.Created.Unix())),
				"created":         strconv.Itoa(int(req.Created.Unix())),
			},
		},
	}

	result, err := s.Client.CoreV1().Namespaces().Create(namespace)
	if err == nil {
		log.Infof("Created namespace: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Namespace %s already exists.", nsName)
		return namespace, err
	} else {
		log.Errorf("Problem creating namespace %s: %s", nsName, err)
		return nil, err
	}
	return result, err
}

// PurgeOldLessons identifies any kubernetes namespaces that are operating with our antidoteId,
// and among those, deletes the ones that have a lastAccessed timestamp that exceeds our configured
// TTL. This function is meant to be run in a loop within a goroutine, at a configured interval. Returns
// a slice of livelesson IDs to be deleted by the caller (not handled by this function)
func (s *AntidoteScheduler) PurgeOldLessons() ([]string, error) {

	nameSpaces, err := s.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll delete way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", s.Config.InstanceID),
	})
	if err != nil {
		return nil, err
	}

	// No need to GC if no matching namespaces exist
	if len(nameSpaces.Items) == 0 {
		log.Debug("No namespaces with our ID found. No need to GC.")
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

		// TODO(mierdin): Gracefully handle this
		lsID := nameSpaces.Items[n].ObjectMeta.Labels["liveSession"]
		ls, err := s.Db.GetLiveSession(lsID)
		if err != nil {
			return []string{}, err
		}
		if ls.Persistent {
			log.Debugf("Skipping GC of expired namespace %s because its sessionId %s is marked as persistent.", nameSpaces.Items[n].Name, ls.ID)
			continue
		}

		liveLessonsToDelete = append(liveLessonsToDelete, nameSpaces.Items[n].ObjectMeta.Labels["liveLesson"])
		oldNameSpaces = append(oldNameSpaces, nameSpaces.Items[n].ObjectMeta.Name)
	}

	// No need to GC if no old namespaces exist
	if len(oldNameSpaces) == 0 {
		log.Debug("No old namespaces found. No need to GC.")
		return []string{}, nil
	}

	log.Infof("Garbage-collecting %d old lessons", len(oldNameSpaces))
	log.Debug(oldNameSpaces)
	var wg sync.WaitGroup
	wg.Add(len(oldNameSpaces))
	for n := range oldNameSpaces {
		go func(ns string) {
			defer wg.Done()
			s.deleteNamespace(ns)
		}(oldNameSpaces[n])
	}
	wg.Wait()
	log.Infof("Finished garbage-collecting %d old lessons", len(oldNameSpaces))

	return liveLessonsToDelete, nil

}

// generateNamespaceName is a helper function for determining the name of our kubernetes
// namespaces, so we don't have to do this all over the codebase and maybe get it wrong.
func generateNamespaceName(antidoteId, liveLessonID string) string {
	return fmt.Sprintf("%s-%s", antidoteId, liveLessonID)
}
