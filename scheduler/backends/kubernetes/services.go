package kubernetes

import (
	"errors"
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

func (k *KubernetesBackend) createService(sc ot.SpanContext, pod *corev1.Pod, req services.LessonScheduleRequest) (*corev1.Service, error) {
	span := ot.StartSpan("kubernetes_service_create", ot.ChildOf(sc))
	defer span.Finish()

	// We want to use the same name as the Pod object, since the service name will be what users try to reach
	// (i.e. use "vqfx1" instead of "vqfx1-svc" or something like that.)
	serviceName := pod.ObjectMeta.Name

	nsName := services.NewUULLID(k.Config.InstanceID, req.LiveLessonID).ToString()

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

	result, err := k.Client.CoreV1().Services(nsName).Create(svc)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	// This is a corner case that has yet to occur (the svc creation taking place without having a clusterIP
	// assigned) but it's easy for us to just check for it really quick, and this is important to properly set
	// up the liveendpoints later.
	if result.Spec.ClusterIP == "" {
		err = errors.New("Service was created but no ClusterIP was assigned")
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	return result, err
}
