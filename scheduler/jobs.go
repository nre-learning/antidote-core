package scheduler

import (
	"fmt"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
)

func (ls *LessonScheduler) killAllJobs(nsName string) error {

	batchclient, err := batchv1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	result, err := batchclient.Jobs(nsName).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to list Jobs: %s", err)
		return err
	}

	existingJobs := result.Items

	for i := range existingJobs {
		err = batchclient.Jobs(nsName).Delete(existingJobs[i].ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	// Block until the jobs are cleaned up, so we don't cause a race condition when the scheduler moves forward with trying to create new jobs
	for {
		//TODO(mierdin): add timeout
		time.Sleep(time.Second * 5)
		result, err = batchclient.Jobs(nsName).List(metav1.ListOptions{})
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

	nsName := fmt.Sprintf("%d-%s-ns", req.LessonDef.LessonID, req.Session)

	batchclient, err := batchv1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	result, err := batchclient.Jobs(nsName).Get(job.Name, metav1.GetOptions{})
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

func (ls *LessonScheduler) configureDevice(ep *pb.Endpoint, req *LessonScheduleRequest) (*batchv1.Job, error) {

	batchclient, err := batchv1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	nsName := fmt.Sprintf("%d-%s-ns", req.LessonDef.LessonID, req.Session)

	jobName := fmt.Sprintf("config-%s", ep.Name)
	podName := fmt.Sprintf("config-%s", ep.Name)

	configJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonID),
				"sessionId":      req.Session,
				"syringeManaged": "yes",
				"stageId":        strconv.Itoa(int(req.Stage)),
			},
		},

		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: nsName,
					Labels: map[string]string{
						"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonID),
						"sessionId":      req.Session,
						"syringeManaged": "yes",
						"stageId":        strconv.Itoa(int(req.Stage)),
					},
				},
				Spec: corev1.PodSpec{

					InitContainers: []corev1.Container{
						{
							Name:  "git-clone",
							Image: "alpine/git",
							Command: []string{
								"/usr/local/git/git-clone.sh",
							},
							Args: []string{
								ls.SyringeConfig.LessonRepoRemote,
								ls.SyringeConfig.LessonRepoBranch,
								ls.SyringeConfig.LessonRepoDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "git-clone",
									ReadOnly:  false,
									MountPath: "/usr/local/git",
								},
								{
									Name:      "git-volume",
									ReadOnly:  false,
									MountPath: ls.SyringeConfig.LessonRepoDir,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "napalm",
							Image: "antidotelabs/napalm",
							Command: []string{
								"napalm",
								"--user=root",
								"--password=VR-netlab9",
								"--vendor=junos",
								fmt.Sprintf("--optional_args=port=%d", ep.Port),
								ep.Host,
								"configure",
								// req.LessonDef.Stages[req.Stage].Configs[ep.Name],
								fmt.Sprintf("/antidote/lessons/lesson-%d/stage%d/configs/%s.txt", req.LessonDef.LessonID, req.Stage, ep.Name),
								"--strategy=merge",
							},

							// TODO(mierdin): ONLY for test/dev. Should re-evaluate for prod
							ImagePullPolicy: "Always",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "git-volume",
									ReadOnly:  false,
									MountPath: ls.SyringeConfig.LessonRepoDir,
								},
							},
						},
					},
					RestartPolicy: "Never",
					Volumes: []corev1.Volume{
						{
							Name: "git-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "git-clone",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "git-clone",
									},
									DefaultMode: &defaultGitFileMode,
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := batchclient.Jobs(nsName).Create(configJob)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created job: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Job %s already exists.", jobName)

		result, err := batchclient.Jobs(nsName).Get(jobName, metav1.GetOptions{})
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
