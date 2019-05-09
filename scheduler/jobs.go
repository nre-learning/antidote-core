package scheduler

import (
	"fmt"
	"strconv"
	"time"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	log "github.com/sirupsen/logrus"

	// Kubernetes types
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ls *LessonScheduler) killAllJobs(nsName, jobType string) error {

	result, err := ls.Client.BatchV1().Jobs(nsName).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("jobType=%s", jobType),
	})
	if err != nil {
		log.Errorf("Unable to list Jobs: %s", err)
		return err
	}

	existingJobs := result.Items

	for i := range existingJobs {
		err = ls.Client.BatchV1().Jobs(nsName).Delete(existingJobs[i].ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	// Block until the jobs are cleaned up, so we don't cause a race condition when the scheduler moves forward with trying to create new jobs
	for {
		//TODO(mierdin): add timeout
		time.Sleep(time.Second * 5)
		result, err = ls.Client.BatchV1().Jobs(nsName).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("jobType=%s", jobType),
		})
		if err != nil {
			log.Errorf("Unable to list Jobs: %s", err)
			return err
		}
		if len(result.Items) == 0 {
			break
		}
	}

	return nil
}

func (ls *LessonScheduler) isCompleted(job *batchv1.Job, req *LessonScheduleRequest) (bool, error) {

	nsName := fmt.Sprintf("%s-ns", req.Uuid)

	result, err := ls.Client.BatchV1().Jobs(nsName).Get(job.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Couldn't retrieve job %s for status update: %s", job.Name, err)
		return false, err
	}
	// https://godoc.org/k8s.io/api/batch/v1#JobStatus
	log.WithFields(log.Fields{
		"jobName":    result.Name,
		"active":     result.Status.Active,
		"successful": result.Status.Succeeded,
		"failed":     result.Status.Failed,
	}).Info("Job Status")

	if result.Status.Failed > 0 {
		log.Errorf("Problem configuring with %s", result.Name)

		//TODO(mierdin): need to count N failures, then when exceeded, surface this back up the channel, to the API, and to the user, so they're not waiting forever.
	}

	// If we call this too quickly, k8s won't have a chance to schedule the pods yet, and the final
	// conditional will return true. So let's also check to see if failed or successful is 0
	// TODO(mierdin): Should also return error if Failed jobs is not 0
	if result.Status.Active == 0 && result.Status.Failed == 0 && result.Status.Succeeded == 0 {
		return false, nil
	}

	return (result.Status.Active == 0), nil

}

func (ls *LessonScheduler) configureDevice(ep *pb.LiveEndpoint, req *LessonScheduleRequest) (*batchv1.Job, error) {

	nsName := fmt.Sprintf("%s-ns", req.Uuid)

	jobName := fmt.Sprintf("config-%s", ep.GetName())
	podName := fmt.Sprintf("config-%s", ep.GetName())

	volumes, volumeMounts, initContainers := ls.getVolumesConfiguration(req.Lesson)

	// configFile := fmt.Sprintf("%s/lessons/lesson-%d/stage%d/configs/%s.txt", ls.SyringeConfig.CurriculumDir, req.Lesson.LessonId, req.Stage, ep.Name)
	configFile := fmt.Sprintf("%s/stage%d/configs/%s.txt", ls.SyringeConfig.CurriculumDir, req.Stage, ep.Name)

	configJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.Lesson.LessonId),
				"syringeManaged": "yes",
				"jobType":        "config",
				"stageId":        strconv.Itoa(int(req.Stage)),
			},
		},

		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: nsName,
					Labels: map[string]string{
						"lessonId":       fmt.Sprintf("%d", req.Lesson.LessonId),
						"syringeManaged": "yes",
						"configPod":      "yes",
						"stageId":        strconv.Itoa(int(req.Stage)),
					},
				},
				Spec: corev1.PodSpec{

					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  "configurator",
							Image: "antidotelabs/configurator",
							Command: []string{
								"/configure.py",
								"antidote",
								"antidotepassword",
								"junos",
								strconv.Itoa(int(ep.Port)),
								ep.Host,
								configFile,
							},

							// TODO(mierdin): ONLY for test/dev. Should re-evaluate for prod
							ImagePullPolicy: "Always",
							VolumeMounts:    volumeMounts,
						},
					},
					RestartPolicy: "Never",
					Volumes:       volumes,
				},
			},
		},
	}

	result, err := ls.Client.BatchV1().Jobs(nsName).Create(configJob)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created job: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Job %s already exists.", jobName)

		result, err := ls.Client.BatchV1().Jobs(nsName).Get(jobName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve job after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating job %s: %s", jobName, err)
		return nil, err
	}
	return result, err
}

func (ls *LessonScheduler) verifyLiveLesson(req *LessonScheduleRequest) (*batchv1.Job, error) {

	nsName := fmt.Sprintf("%s-ns", req.Uuid)

	jobName := fmt.Sprintf("verify-%d-%d", req.Lesson.LessonId, req.Stage)
	podName := fmt.Sprintf("verify-%d-%d", req.Lesson.LessonId, req.Stage)

	var retry int32 = 1

	volumes, volumeMounts, initContainers := ls.getVolumesConfiguration(req.Lesson)

	verifyJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.Lesson.LessonId),
				"syringeManaged": "yes",
				"jobType":        "verify",
				"stageId":        strconv.Itoa(int(req.Stage)),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &retry,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: nsName,
					Labels: map[string]string{
						"lessonId":       fmt.Sprintf("%d", req.Lesson.LessonId),
						"syringeManaged": "yes",
						"verifyPod":      "yes",
						"stageId":        strconv.Itoa(int(req.Stage)),
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  "verifier",
							Image: "antidotelabs/utility",
							Command: []string{
								"python",
								fmt.Sprintf("/antidote/lessons/lesson-%d/stage%d/verify.py", req.Lesson.LessonId, req.Stage),
							},

							// TODO(mierdin): ONLY for test/dev. Should re-evaluate for prod
							ImagePullPolicy: "Always",
							VolumeMounts:    volumeMounts,
						},
					},
					RestartPolicy: "Never",
					Volumes:       volumes,
				},
			},
		},
	}

	// if config.SkipLessonClone, use read-only volume mount to local filesystem?

	result, err := ls.Client.BatchV1().Jobs(nsName).Create(verifyJob)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created job: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Job %s already exists.", jobName)

		result, err := ls.Client.BatchV1().Jobs(nsName).Get(jobName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve job after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating job %s: %s", jobName, err)
		return nil, err
	}
	return result, err
}

func (ls *LessonScheduler) verifyStatus(job *batchv1.Job, req *LessonScheduleRequest) (finished bool, err error) {

	nsName := fmt.Sprintf("%s-ns", req.Uuid)

	result, err := ls.Client.BatchV1().Jobs(nsName).Get(job.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Couldn't retrieve job %s for status update: %s", job.Name, err)
		return false, err
	}
	// https://godoc.org/k8s.io/api/batch/v1#JobStatus
	log.WithFields(log.Fields{
		"jobName":    result.Name,
		"active":     result.Status.Active,
		"successful": result.Status.Succeeded,
		"failed":     result.Status.Failed,
	}).Info("Job Status")

	if result.Status.Failed > 0 {
		log.Errorf("Problem verifying with %s", result.Name)
		return true, fmt.Errorf("Problem verifying with %s", result.Name)
	}

	// If we call this too quickly, k8s won't have a chance to schedule the pods yet, and the final
	// conditional will return true. So let's also check to see if failed or successful is 0
	if result.Status.Active == 0 && result.Status.Failed == 0 && result.Status.Succeeded == 0 {
		return false, nil
	}

	return (result.Status.Active == 0), nil

}
