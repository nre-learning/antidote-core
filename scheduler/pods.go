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

	volumes, volumeMounts, initContainers := s.getVolumesConfiguration(span.Context(), req.LessonSlug)

	privileged := false

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
		privileged = image.Privileged
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

	// Not all endpoint images come with a hypervisor. For these, we want to be able to conditionally
	// enable/disable privileged mode based on the relevant field present in the loaded image spec.
	//
	// See https://github.com/nre-learning/proposals/pull/7 for plans to standardize the endpoint image
	// build process, which is likely to include a virtualization layer for safety. However, this option will
	// likely remain in place regardless.
	if privileged {
		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged:               &privileged,
			AllowPrivilegeEscalation: &privileged,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
		}
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
// If a pod starts successfully but fails after (resulting in a CrashLoopBackoff state) then this
// will likely not catch it. The next network-centric health checks should handle that
func (s *AntidoteScheduler) getPodStatus(origPod *corev1.Pod) (bool, error) {

	/*
		The logic here is as follows:

		- return false and an error if we encounter some kind of failure state either in the pod or in getting the pod
		- return false and no error if we think things are still starting
		- return true and no error if we think everything is ready to go
	*/

	pod, err := s.Client.CoreV1().Pods(origPod.ObjectMeta.Namespace).Get(origPod.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {

		return false, err
	}

	// Note that this doesn't cover init container failures, nor does it cover post-Ready failures.
	if pod.Status.Phase == corev1.PodFailed {
		return false, errors.New("Pod in failure state")
	}

	if pod.Status.Phase == corev1.PodRunning {
		return true, nil
	}

	return false, nil
}

func (s *AntidoteScheduler) getPodLogs(pod *corev1.Pod, container string) string {

	var plo = corev1.PodLogOptions{}
	if container != "" {
		plo.Container = container
	}
	req := s.Client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &plo)
	podLogs, err := req.Stream()
	if err != nil {
		return "error in opening stream"
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf"
	}
	str := buf.String()

	return str
}
