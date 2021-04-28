package kubernetes

import (
	"testing"

	"github.com/jinzhu/copier"
	"github.com/nre-learning/antidote-core/services"
	ot "github.com/opentracing/opentracing-go"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestJobs(t *testing.T) {

	req := services.LessonScheduleRequest{
		LiveLessonID: "asdf",
		LessonSlug:   "test-lesson",
	}

	k := createFakeKubernetesBackend()
	nsName := generateNamespaceName(k.Config.InstanceID, req.LiveLessonID)

	jobName := "configjob"
	span := ot.StartSpan("test_db")
	defer span.Finish()

	backoff := int32(JobBackoff)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: nsName,
			Labels: map[string]string{
				"antidoteManaged": "yes",
				"jobType":         "config",
			},
		},

		Spec: batchv1.JobSpec{
			BackoffLimit: &backoff,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: nsName,
					Labels: map[string]string{
						"antidoteManaged": "yes",
						"configPod":       "yes",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "configurator",
							Image:           "antidotelabs/configurator",
							Command:         []string{"foo"},
							ImagePullPolicy: v1.PullAlways,
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}

	_, err := k.Client.BatchV1().Jobs(nsName).Create(job)
	ok(t, err)
	result, err := k.Client.BatchV1().Jobs(nsName).Get(job.Name, metav1.GetOptions{})
	ok(t, err)
	equals(t, result.Namespace, "antidote-testing-asdf")
	err = k.Client.BatchV1().Jobs(nsName).Delete(job.Name, &metav1.DeleteOptions{})
	ok(t, err)

	// A new job with no status should return false with no error
	t.Run("", func(t *testing.T) {
		_ = k.Client.BatchV1().Jobs(nsName).Delete(job.Name, &metav1.DeleteOptions{})
		_, err := k.Client.BatchV1().Jobs(nsName).Create(job)
		ok(t, err)

		completed, statusCount, err := k.getJobStatus(span, job, req)
		ok(t, err)
		equals(t, false, completed)
		equals(t, map[string]int32{"active": 0, "failed": 0, "succeeded": 0}, statusCount)
	})

	// A job with at least one success should return true and no error
	t.Run("", func(t *testing.T) {
		_ = k.Client.BatchV1().Jobs(nsName).Delete(job.Name, &metav1.DeleteOptions{})
		jobcopy := &batchv1.Job{}
		copier.Copy(&jobcopy, &job)
		jobcopy.Status.Succeeded = 1
		_, err := k.Client.BatchV1().Jobs(nsName).Create(jobcopy)
		ok(t, err)

		completed, statusCount, err := k.getJobStatus(span, job, req)
		ok(t, err)
		equals(t, true, completed)
		equals(t, map[string]int32{"active": 0, "failed": 0, "succeeded": 1}, statusCount)
	})

	// A job with a number of failures that is less than the backoff limit should return false with no error
	t.Run("", func(t *testing.T) {
		_ = k.Client.BatchV1().Jobs(nsName).Delete(job.Name, &metav1.DeleteOptions{})
		jobcopy := &batchv1.Job{}
		copier.Copy(&jobcopy, &job)
		jobcopy.Status.Failed = 2
		_, err := k.Client.BatchV1().Jobs(nsName).Create(jobcopy)
		ok(t, err)

		completed, statusCount, err := k.getJobStatus(span, job, req)
		ok(t, err)
		equals(t, false, completed)
		equals(t, map[string]int32{"active": 0, "failed": 2, "succeeded": 0}, statusCount)
	})

	// A job with a number of failures that is equal to or greater than the backoff limit should return true with an error
	t.Run("", func(t *testing.T) {
		_ = k.Client.BatchV1().Jobs(nsName).Delete(job.Name, &metav1.DeleteOptions{})
		jobcopy := &batchv1.Job{}
		copier.Copy(&jobcopy, &job)
		jobcopy.Status.Failed = 3
		_, err := k.Client.BatchV1().Jobs(nsName).Create(jobcopy)
		ok(t, err)

		completed, statusCount, err := k.getJobStatus(span, job, req)
		equals(t, true, completed)
		assert(t, (err != nil), "")
		equals(t, map[string]int32{"active": 0, "failed": 3, "succeeded": 0}, statusCount)
	})

	// A job with an improper namespace should cause a failure with all status count set to 0
	// and a completed status of "true", just to indicate we shouldn't keep trying
	t.Run("", func(t *testing.T) {
		_ = k.Client.BatchV1().Jobs(nsName).Delete(job.Name, &metav1.DeleteOptions{})
		jobcopy := &batchv1.Job{}
		copier.Copy(&jobcopy, &job)
		jobcopy.Namespace = "foobar"
		_, err := k.Client.BatchV1().Jobs("foobar").Create(jobcopy)
		ok(t, err)

		completed, statusCount, err := k.getJobStatus(span, job, req)
		equals(t, true, completed)
		assert(t, (err != nil), "")
		equals(t, map[string]int32{"active": 0, "failed": 0, "succeeded": 0}, statusCount)
	})

}
