package scheduler

import (
	"fmt"

	"github.com/nre-learning/antidote-core/services"
	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	// Kubernetes types
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (s *AntidoteScheduler) createService(sc ot.SpanContext, pod *corev1.Pod, req services.LessonScheduleRequest) (*corev1.Service, error) {
	span := ot.StartSpan("scheduler_service_create", ot.ChildOf(sc))
	defer span.Finish()

	// We want to use the same name as the Pod object, since the service name will be what users try to reach
	// (i.e. use "vqfx1" instead of "vqfx1-svc" or something like that.)
	serviceName := pod.ObjectMeta.Name

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nsName,
			Labels: map[string]string{
				"liveLesson":      fmt.Sprintf("%s", req.LiveLessonID),
				"liveSession":     fmt.Sprintf("%s", req.LiveSessionID),
				"antidoteManaged": "yes",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"liveLesson":  fmt.Sprintf("%s", req.LiveLessonID),
				"liveSession": fmt.Sprintf("%s", req.LiveSessionID),
				"podName":     pod.ObjectMeta.Name,
			},
			Ports: []corev1.ServicePort{}, // will fill out below

			Type: corev1.ServiceTypeClusterIP,
		},
	}

	for p := range pod.Spec.Containers[0].Ports {

		port := pod.Spec.Containers[0].Ports[p].ContainerPort

		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
			Name:       fmt.Sprintf("port-%d", port),
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
		})
	}

	// TODO(mierdin): The code that calls this function relies on svc.Spec.ClusterIP
	// to set the Host property for the corresponding LiveEndpoint. It appears that this information
	// is provided from the kubernetes client function below, but I'm not sure how reliable that is.
	// It has proven acceptably reliable thus far, but might be worth looking at, and perhaps adding a quick
	// check that this information is present before returning
	result, err := s.Client.CoreV1().Services(nsName).Create(svc)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	return result, err
}
