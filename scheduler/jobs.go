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
	logrus "github.com/sirupsen/logrus"

	// Kubernetes types
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This constant controls the number of times a job is retried before we consider it failed.
const JobBackoff = 2

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
	if len(existingJobs) == 0 {
		return nil
	}

	for i := range existingJobs {
		err = s.Client.BatchV1().Jobs(nsName).Delete(existingJobs[i].ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	// Block until the jobs are cleaned up, so we don't cause a race
	// condition when the scheduler moves forward with trying to create new jobs
	for i := 0; i < 60; i++ {
		result, err = s.Client.BatchV1().Jobs(nsName).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("jobType=%s", jobType),
		})
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}
		if len(result.Items) == 0 {
			return nil
		}

		time.Sleep(time.Second * 1)
	}

	err = errors.New("Timed out waiting for old jobs to be cleaned up")
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return err
}

func (s *AntidoteScheduler) getJobStatus(span ot.Span, job *batchv1.Job, req services.LessonScheduleRequest) (bool, map[string]int32, error) {

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	result, err := s.Client.BatchV1().Jobs(nsName).Get(job.Name, metav1.GetOptions{})
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return false,
			map[string]int32{
				"active":    result.Status.Active,
				"succeeded": result.Status.Succeeded,
				"failed":    result.Status.Failed,
			},
			err
	}

	if result.Status.Failed >= JobBackoff+1 {

		// Get logs for failed configuration job/pod for troubleshooting purposes later
		pods, err := s.Client.CoreV1().Pods(nsName).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		})
		if err != nil || len(pods.Items) == 0 {
			logrus.Debugf("Unable to retrieve logs for failed configuration pod in livelesson %s", req.LiveLessonID)
		} else {
			failedLogs := s.getPodLogs(&pods.Items[len(pods.Items)-1])
			span.LogEventWithPayload("jobFailureLogs", services.SafePayload(failedLogs))
		}

		// Log error to span and return
		err = fmt.Errorf("Too many failures when trying to configure %s", result.Name)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return true,
			map[string]int32{
				"active":    result.Status.Active,
				"succeeded": result.Status.Succeeded,
				"failed":    result.Status.Failed,
			},
			err
	}

	// If we call this too quickly, k8s won't have a chance to schedule the pods yet, and the final
	// conditional will return true. So let's also check to see if failed or successful is 0
	// TODO(mierdin): Should also return error if Failed jobs is not 0
	if result.Status.Active == 0 && result.Status.Failed == 0 && result.Status.Succeeded == 0 {
		return false,
			map[string]int32{
				"active":    result.Status.Active,
				"succeeded": result.Status.Succeeded,
				"failed":    result.Status.Failed,
			},
			nil
	}

	return (result.Status.Active == 0), map[string]int32{
		"active":    result.Status.Active,
		"succeeded": result.Status.Succeeded,
		"failed":    result.Status.Failed,
	}, nil

}

func (s *AntidoteScheduler) configureEndpoint(sc ot.SpanContext, ep *models.LiveEndpoint, req services.LessonScheduleRequest) (*batchv1.Job, error) {
	span := ot.StartSpan("scheduler_configure_endpoint", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("endpointName", ep.Name)

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	jobName := fmt.Sprintf("config-%s-%d", ep.Name, req.Stage)
	podName := fmt.Sprintf("config-%s-%d", ep.Name, req.Stage)

	volumes, volumeMounts, initContainers := s.getVolumesConfiguration(span.Context(), req.LessonSlug)

	image, err := s.Db.GetImage(span.Context(), ep.Image)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	var configCommand []string
	configFilePath := fmt.Sprintf("/antidote/stage%d/configs/%s", req.Stage, ep.ConfigurationFile)

	if ep.ConfigurationType == "python" {
		configCommand = []string{
			"python",
			configFilePath,
		}
	} else if ep.ConfigurationType == "ansible" {
		configCommand = []string{
			"ansible-playbook",
			"-vvvv",
			"-i",
			fmt.Sprintf("%s,", ep.Host),
			configFilePath,
		}
	} else if ep.ConfigurationType == "napalm" {

		// determine NAPALM driver from filename. Structure is:
		// <endpoint>-<napalmDriver>.txt. This is enforced on ingest, so as
		// long as we do basic sanity checks on how many results we get from a split
		// using . and - as delimiters, we should be okay.
		separated := strings.FieldsFunc(configFilePath, func(r rune) bool {
			return r == '-' || r == '.'
		})
		if len(separated) < 3 {
			return nil, errors.New("Invalid napalm driver string")
		}

		napalmDriver := separated[1]
		configCommand = []string{
			"/configure.py",
			image.ConfigUser,
			image.ConfigPassword,
			napalmDriver,
			"22",
			ep.Host,
			configFilePath,
		}
	} else {
		return nil, errors.New("Unknown config type")
	}

	pullPolicy := v1.PullIfNotPresent
	if s.Config.AlwaysPull {
		pullPolicy = v1.PullAlways
	}

	backoff := int32(JobBackoff)
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
			BackoffLimit: &backoff,
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
							Name:            "configurator",
							Image:           fmt.Sprintf("antidotelabs/configurator:%s", s.BuildInfo["imageVersion"]),
							Command:         configCommand,
							ImagePullPolicy: pullPolicy,
							Env: []corev1.EnvVar{
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
