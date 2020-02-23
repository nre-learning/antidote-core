package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	// Kubernetes types
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ls *LessonScheduler) boopNamespace(nsName string) error {

	log.Debugf("Booping %s", nsName)

	ns, err := ls.Client.CoreV1().Namespaces().Get(nsName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ns.ObjectMeta.Labels["lastAccessed"] = strconv.Itoa(int(time.Now().Unix()))

	_, err = ls.Client.CoreV1().Namespaces().Update(ns)
	if err != nil {
		return err
	}

	// "syringeManaged": "yes",

	return nil
}

// cleanOrphanedNamespaces seeks out all syringe-managed namespaces, and deletes them.
// This will effectively reset the cluster to a state with all of the remaining infrastructure
// in place, but no running lessons. Syringe doesn't manage itself, or any other Antidote services.
func (ls *LessonScheduler) cleanOrphanedNamespaces() error {

	nameSpaces, err := ls.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll nuke way more than you intended
		LabelSelector: fmt.Sprintf("syringeManaged=yes,syringeId=%s", ls.SyringeConfig.SyringeID),
	})
	if err != nil {
		return err
	}

	// No need to nuke if no syringe namespaces exist
	if len(nameSpaces.Items) == 0 {
		log.Info("No namespaces with our syringeId found. Starting normally.")
		return nil
	}

	log.Warnf("Nuking all namespaces with a syringeId of %s", ls.SyringeConfig.SyringeID)
	var wg sync.WaitGroup
	wg.Add(len(nameSpaces.Items))
	for n := range nameSpaces.Items {

		nsName := nameSpaces.Items[n].ObjectMeta.Name
		go func() {
			defer wg.Done()
			ls.deleteNamespace(nsName)
		}()
	}
	wg.Wait()
	log.Info("Nuke complete. It was the only way to be sure...")
	return nil
}

func (ls *LessonScheduler) deleteNamespace(name string) error {

	err := ls.Client.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	// Wait for the namespace to be deleted
	deleteTimeoutSecs := 120
	for i := 0; i < deleteTimeoutSecs/5; i++ {
		time.Sleep(5 * time.Second)

		_, err := ls.Client.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
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

func (ls *LessonScheduler) createNamespace(req *LessonScheduleRequest) (*corev1.Namespace, error) {

	nsName := generateNamespaceName(ls.SyringeConfig.SyringeID, req.LiveLessonID)

	log.Infof("Creating namespace: %s", nsName)

	ll, err := ls.Db.GetLiveLesson(req.LiveLessonID)
	if err != nil {
		return nil, err
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"liveLesson":     fmt.Sprintf("%s", req.LiveLessonID),
				"liveSession":    fmt.Sprintf("%s", req.LiveSessionID),
				"lessonSlug":     fmt.Sprintf("%s", ll.LessonSlug),
				"syringeManaged": "yes",
				"syringeId":      ls.SyringeConfig.SyringeID,
				"lastAccessed":   strconv.Itoa(int(req.Created.Unix())),
				"created":        strconv.Itoa(int(req.Created.Unix())),
			},
		},
	}

	result, err := ls.Client.CoreV1().Namespaces().Create(namespace)
	if err == nil {
		log.Infof("Created namespace: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Namespace %s already exists.", nsName)

		// In this case we are returning what we tried to create. This means that when this lesson is cleaned up,
		// syringe will delete the pod that already existed.
		return namespace, err
	} else {
		log.Errorf("Problem creating namespace %s: %s", nsName, err)
		return nil, err
	}
	return result, err
}

// PurgeOldLessons identifies any kubernetes namespaces that are operating with our syringeId,
// and among those, deletes the ones that have a lastAccessed timestamp that exceeds our configured
// TTL. This function is meant to be run in a loop within a goroutine, at a configured interval.
func (ls *LessonScheduler) PurgeOldLessons() ([]string, error) {

	nameSpaces, err := ls.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll delete way more than you intended
		LabelSelector: fmt.Sprintf("syringeManaged=yes,syringeId=%s", ls.SyringeConfig.SyringeID),
	})
	if err != nil {
		return nil, err
	}

	// No need to GC if no matching namespaces exist
	if len(nameSpaces.Items) == 0 {
		log.Debug("No namespaces with our ID found. No need to GC.")
		return []string{}, nil
	}

	oldNameSpaces := []string{}
	for n := range nameSpaces.Items {

		// lastAccessed =
		i, err := strconv.ParseInt(nameSpaces.Items[n].ObjectMeta.Labels["lastAccessed"], 10, 64)
		if err != nil {
			return nil, err
		}
		lastAccessed := time.Unix(i, 0)

		if time.Since(lastAccessed) < time.Duration(ls.SyringeConfig.LiveLessonTTL)*time.Minute {
			continue
		}

		ls, err := ls.Db.GetLiveSession(nameSpaces.Items[n].ObjectMeta.Labels["liveSession"])
		if err != nil {
			return nil, err
		}
		if ls.Persistent {
			log.Debugf("Skipping GC of expired namespace %s because its sessionId %s is marked as persistent.", nameSpaces.Items[n].Name, ls.ID)
			continue
		}

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
			ls.deleteNamespace(ns)
		}(oldNameSpaces[n])
	}
	wg.Wait()
	log.Infof("Finished garbage-collecting %d old lessons", len(oldNameSpaces))
	return oldNameSpaces, nil

}

// generateNamespaceName is a helper function for determining the name of our kubernetes
// namespaces, so we don't have to do this all over the codebase and maybe get it wrong.
func generateNamespaceName(syringeID, liveLessonID string) string {
	return fmt.Sprintf("%s-%s", syringeID, liveLessonID)
}
