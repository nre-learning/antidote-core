package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	// Kubernetes types
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *AntidoteScheduler) killAllJobs(sc ot.SpanContext, nsName, jobType string) error {

	span := ot.StartSpan("scheduler_job_killall", ot.ChildOf(sc))
	defer span.Finish()

	result, err := s.Client.BatchV1().Jobs(nsName).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("jobType=%s", jobType),
	})
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	existingJobs := result.Items

	for i := range existingJobs {
		err = s.Client.BatchV1().Jobs(nsName).Delete(existingJobs[i].ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	// Block until the jobs are cleaned up, so we don't cause a race condition when the scheduler moves forward with trying to create new jobs
	for {
		//TODO(mierdin): add timeout
		time.Sleep(time.Second * 5)
		result, err = s.Client.BatchV1().Jobs(nsName).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("jobType=%s", jobType),
		})
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}
		if len(result.Items) == 0 {
			break
		}
	}

	return nil
}

func (s *AntidoteScheduler) isCompleted(span ot.Span, job *batchv1.Job, req services.LessonScheduleRequest) (bool, error) {

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	result, err := s.Client.BatchV1().Jobs(nsName).Get(job.Name, metav1.GetOptions{})
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return false, err
	}
	// https://godoc.org/k8s.io/api/batch/v1#JobStatus
	span.LogFields(
		log.String("jobName", result.Name),
		log.Int32("active", result.Status.Active),
		log.Int32("successful", result.Status.Succeeded),
		log.Int32("failed", result.Status.Failed),
	)

	if result.Status.Failed >= 3 {
		err = fmt.Errorf("Too many failures when trying to configure %s", result.Name)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return true, err
	}

	// If we call this too quickly, k8s won't have a chance to schedule the pods yet, and the final
	// conditional will return true. So let's also check to see if failed or successful is 0
	// TODO(mierdin): Should also return error if Failed jobs is not 0
	if result.Status.Active == 0 && result.Status.Failed == 0 && result.Status.Succeeded == 0 {
		return false, nil
	}

	return (result.Status.Active == 0), nil

}

func (s *AntidoteScheduler) configureEndpoint(sc ot.SpanContext, ep *models.LiveEndpoint, req services.LessonScheduleRequest) (*batchv1.Job, error) {
	span := ot.StartSpan("scheduler_configure_endpoint", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	jobName := fmt.Sprintf("config-%s-%d", ep.Name, req.Stage)
	podName := fmt.Sprintf("config-%s-%d", ep.Name, req.Stage)

	volumes, volumeMounts, initContainers := s.getVolumesConfiguration(span.Context(), req.LessonSlug)

	var configCommand []string

	if ep.ConfigurationType == "python" {
		configCommand = []string{
			"python",
			fmt.Sprintf("/antidote/stage%d/configs/%s.py", req.Stage, ep.Name),
		}
	} else if ep.ConfigurationType == "ansible" {
		configCommand = []string{
			"ansible-playbook",
			"-vvvv",
			"-i",
			fmt.Sprintf("%s,", ep.Host),
			fmt.Sprintf("/antidote/stage%d/configs/%s.yml", req.Stage, ep.Name),
		}
	} else if strings.HasPrefix(ep.ConfigurationType, "napalm") {

		separated := strings.Split(ep.ConfigurationType, "-")
		if len(separated) < 2 {
			return nil, errors.New("Invalid napalm driver string")
		}
		configCommand = []string{
			"/configure.py",
			"antidote",
			"antidotepassword",
			separated[1],
			"22",
			ep.Host,
			fmt.Sprintf("/antidote/stage%d/configs/%s.txt", req.Stage, ep.Name),
		}
	} else {
		return nil, errors.New("Unknown config type")
	}

	configJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: nsName,
			Labels: map[string]string{
				"antidoteManaged": "yes",
				"jobType":         "config",
				"stageId":         strconv.Itoa(int(req.Stage)),
			},
		},

		Spec: batchv1.JobSpec{
			// BackoffLimit: int32(3),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: nsName,
					Labels: map[string]string{
						"antidoteManaged": "yes",
						"configPod":       "yes",
						"stageId":         strconv.Itoa(int(req.Stage)),
					},
				},
				Spec: corev1.PodSpec{

					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:    "configurator",
							Image:   fmt.Sprintf("antidotelabs/configurator:%s", s.BuildInfo["imageVersion"]),
							Command: configCommand,

							// TODO(mierdin): ONLY for test/dev. Should re-evaluate for prod
							ImagePullPolicy: "Always",
							Env: []corev1.EnvVar{

								// Providing intended host to configurator
								{Name: "SYRINGE_TARGET_HOST", Value: ep.Host},

								{Name: "ANSIBLE_HOST_KEY_CHECKING", Value: "False"},
							},
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy: "Never",
					Volumes:       volumes,
				},
			},
		},
	}

	result, err := s.Client.BatchV1().Jobs(nsName).Create(configJob)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}
	return result, err
}
