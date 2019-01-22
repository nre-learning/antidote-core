package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LessonScheduler) boopNamespace(nsName string) error {

	log.Debugf("Booping %s", nsName)

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}
	ns, err := coreclient.Namespaces().Get(nsName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ns.ObjectMeta.Labels["lastAccessed"] = strconv.Itoa(int(time.Now().Unix()))

	_, err = coreclient.Namespaces().Update(ns)
	if err != nil {
		return err
	}

	// "syringeManaged": "yes",

	return nil
}

// nukeFromOrbit seeks out all syringe-managed namespaces, and deletes them.
// This will effectively reset the cluster to a state with all of the remaining infrastructure
// in place, but no running lessons. Syringe doesn't manage itself, or any other Antidote services.
func (ls *LessonScheduler) nukeFromOrbit() error {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}
	nameSpaces, err := coreclient.Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll nuke way more than you intended
		LabelSelector: fmt.Sprintf("syringeManaged=yes,syringeTier=%s", ls.SyringeConfig.Tier),
	})
	if err != nil {
		return err
	}

	// No need to nuke if no syringe namespaces exist
	if len(nameSpaces.Items) == 0 {
		log.Info("No syringe-managed namespaces found. Starting normally.")
		return nil
	}

	log.Warn("Nuking all syringe-managed namespaces")
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

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	err = coreclient.Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	// Wait for the namespace to be deleted
	deleteTimeoutSecs := 120
	for i := 0; i < deleteTimeoutSecs/5; i++ {
		time.Sleep(5 * time.Second)

		_, err := coreclient.Namespaces().Get(name, metav1.GetOptions{})
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

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	nsName := fmt.Sprintf("%s-ns", req.Uuid)

	log.Infof("Creating namespace: %s", nsName)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonId),
				"syringeManaged": "yes",
				"name": nsName,
				"syringeTier":    ls.SyringeConfig.Tier,
				"lastAccessed":   strconv.Itoa(int(time.Now().Unix())),
				"created":        strconv.Itoa(int(time.Now().Unix())),
			},
			Namespace: nsName,
		},
	}

	result, err := coreclient.Namespaces().Create(namespace)
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

// Lesson garbage-collector
func (ls *LessonScheduler) purgeOldLessons() ([]string, error) {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}
	nameSpaces, err := coreclient.Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll delete way more than you intended
		LabelSelector: fmt.Sprintf("syringeManaged=yes,syringeTier=%s", ls.SyringeConfig.Tier),
	})
	if err != nil {
		return nil, err
	}

	// No need to GC if no syringe namespaces exist
	if len(nameSpaces.Items) == 0 {
		log.Debug("No syringe-managed namespaces found. No need to GC.")
		return []string{}, nil
	}

	oldNameSpaces := []string{}
	for n := range nameSpaces.Items {

		// lastAccessed =
		i, err := strconv.ParseInt(nameSpaces.Items[n].ObjectMeta.Labels["lastAccessed"], 10, 64)
		if err != nil {
			panic(err)
		}
		lastAccessed := time.Unix(i, 0)
		if time.Since(lastAccessed) < 30*time.Minute {
			continue
		}

		// Skip GC if this session is in whitelist
		session := strings.Split(nameSpaces.Items[n].Name, "-")[1]
		if _, ok := ls.GcWhiteList[session]; ok {
			log.Debugf("Skipping GC of expired namespace %s because this session is in the whitelist.", nameSpaces.Items[n].Name)
			continue
		}

		oldNameSpaces = append(oldNameSpaces, nameSpaces.Items[n].ObjectMeta.Name)
	}

	// No need to GC if no old namespaces exist
	if len(oldNameSpaces) == 0 {
		log.Debug("No old namespaces found. No need to GC.")
		return []string{}, nil
	}

	log.Warnf("Garbage-collecting %d old lessons", len(oldNameSpaces))
	var wg sync.WaitGroup
	wg.Add(len(oldNameSpaces))
	for n := range oldNameSpaces {
		go func() {
			defer wg.Done()
			ls.deleteNamespace(oldNameSpaces[n])
		}()
	}
	wg.Wait()
	log.Infof("Finished garbage-collecting %d old lessons", len(oldNameSpaces))
	return oldNameSpaces, nil

}
