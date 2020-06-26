package scheduler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	// Kubernetes Types
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *AntidoteScheduler) createPod(sc ot.SpanContext, ep *models.LiveEndpoint, networks []string, req services.LessonScheduleRequest) (*corev1.Pod, error) {

	span := ot.StartSpan("scheduler_pod_create", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

	span.SetTag("epName", ep.Name)
	span.SetTag("nsName", nsName)

	type networkAnnotation struct {
		Name string `json:"name"`
	}

	netAnnotations := []networkAnnotation{}
	for n := range networks {
		netAnnotations = append(netAnnotations, networkAnnotation{Name: networks[n]})
	}

	netAnnotationsJSON, err := json.Marshal(netAnnotations)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	volumes, volumeMounts, initContainers, err := s.getVolumesConfiguration(span.Context(), req.LessonSlug)
	if err != nil {
		err := fmt.Errorf("Unable to get volumes configuration: %v", err)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	// privileged := false

	flavor := models.FlavorPlain

	// If the endpoint is a jupyter server, we don't want to append a curriculum version,
	// because that's part of the platform. For all others, we will append the version of the curriculum.
	var imageRef string
	if strings.Contains(ep.Image, "jupyter") {
		imageRef = ep.Image
	} else {

		image, err := s.Db.GetImage(span.Context(), ep.Image)
		if err != nil {
			return nil, fmt.Errorf("Unable to find referenced image %s in data store: %v", ep.Image, err)
		}
		flavor = image.Flavor
		imageRef = fmt.Sprintf("%s/%s:%s", s.Config.ImageOrg, ep.Image, s.Config.CurriculumVersion)
	}

	pullPolicy := v1.PullIfNotPresent
	if s.Config.AlwaysPull {
		pullPolicy = v1.PullAlways
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ep.Name,
			Namespace: nsName,
			Labels: map[string]string{
				"liveLesson":      fmt.Sprintf("%s", req.LiveLessonID),
				"liveSession":     fmt.Sprintf("%s", req.LiveSessionID),
				"podName":         ep.Name,
				"antidoteManaged": "yes",
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": string(netAnnotationsJSON),
			},
		},
		Spec: corev1.PodSpec{

			// All antidote-created pods are assigned to the same host for a given namespace. This keeps things much simplier, since each
			// network just uses linux bridges local to that host. Multi-host networking is a bit hit-or-miss when used with multus, so
			// this just keeps things simpler.
			// https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
			Affinity: &corev1.Affinity{
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"liveLesson":      fmt.Sprintf("%s", req.LiveLessonID),
									"liveSession":     fmt.Sprintf("%s", req.LiveSessionID),
									"antidoteManaged": "yes",
								},
							},
							Namespaces: []string{
								nsName,
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},

			InitContainers: initContainers,
			Containers: []corev1.Container{
				{
					Name:            ep.Name,
					Image:           imageRef,
					ImagePullPolicy: pullPolicy,

					Ports:        []corev1.ContainerPort{}, // Will set below
					VolumeMounts: volumeMounts,
				},
			},

			Volumes: volumes,
		},
	}

	if s.Config.PullCredName != "" {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: s.Config.PullCredName})
	} else {
		span.LogEvent("PullCredsLocation either blank or invalid format, skipping pod attachment")
	}

	// Not all endpoint images come with a hypervisor. For these, we want to be able to conditionally
	// enable/disable privileged mode based on the relevant field present in the loaded image spec.
	//
	// See https://github.com/nre-learning/proposals/pull/7 for plans to standardize the endpoint image
	// build process, which is likely to include a virtualization layer for safety. However, this option will
	// likely remain in place regardless.

	switch flavor {
	// TODO(mierdin): Change this?
	case models.FlavorPrivileged:
		t := true
		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged:               &t,
			AllowPrivilegeEscalation: &t,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
		}
	case models.FlavorPlain:
		// TODO(mierdin): May not keep this
	default:

		// This should enable the kata runtime and NEVER give privileges. This is so we default to a secure position should something fail.
		// The last thing I want to do is grant privileges and forget to install the runtimeclass for kata, which would be bad.
		// TODO(mierdin): even with this discipline, it might be worth looking into performing a startup check within antidote-core
		// for the presence of this runtimeclass (though this would likely require more gross CRD code)
		// (runtimeClassName: kata)
		kata := "kata"
		pod.Spec.RuntimeClassName = &kata
	}

	// Convert to ContainerPort and attach to pod container
	for p := range ep.Ports {
		pod.Spec.Containers[0].Ports = append(pod.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: ep.Ports[p]})
	}

	if len(pod.Spec.Containers[0].Ports) == 0 {
		return nil, fmt.Errorf("not creating pod %s - must have at least one port exposed", pod.ObjectMeta.Name)
	}

	result, err := s.Client.CoreV1().Pods(nsName).Create(pod)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	return result, nil
}

// getPodStatus is a k8s-focused health check. Just a sanity check to ensure the pod is running from
// kubernetes perspective, before we move forward with network-based health checks. It is not meant to capture
// all potential failure scenarios, only those that result in the Pod failing to start in the first place.
func (s *AntidoteScheduler) getPodStatus(origPod *corev1.Pod) (bool, error) {
	pod, err := s.Client.CoreV1().Pods(origPod.ObjectMeta.Namespace).Get(origPod.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	// We expect to see an init container status, so if we don't see one, we know we're not
	// ready yet. Also useful to ensure we don't get an "index out of range" panic
	if len(pod.Status.InitContainerStatuses) == 0 {
		return false, nil
	}

	if pod.Status.InitContainerStatuses[0].State.Terminated != nil {
		if pod.Status.InitContainerStatuses[0].State.Terminated.ExitCode != 0 {
			return false, errors.New("Init container failed")
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		return false, errors.New("Pod in failure state")
	}
	if pod.Status.Phase == corev1.PodRunning {
		return true, nil
	}

	return false, nil
}

// recordPodLogs allows us to record the logs for a given pod (typically as a result of a failure of some kind),
// such as an endpoint pod or a pod spawned by a Job during configuration. These logs are retrieved,
// and then exported via a dedicated OpenTracing span.
func (s *AntidoteScheduler) recordPodLogs(sc ot.SpanContext, llID, podName string, container string) {
	span := ot.StartSpan("scheduler_pod_logs", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("podName", podName)
	span.SetTag("container", container)

	nsName := generateNamespaceName(s.Config.InstanceID, llID)
	pod, err := s.Client.CoreV1().Pods(nsName).Get(podName, metav1.GetOptions{})
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	var plo = corev1.PodLogOptions{}
	if container != "" {
		plo.Container = container
	}
	req := s.Client.CoreV1().Pods(nsName).GetLogs(pod.Name, &plo)
	podLogs, err := req.Stream()
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}
	str := buf.String()

	span.LogEventWithPayload("logs", services.SafePayload(str))
}
