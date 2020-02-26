package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"

	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
	log "github.com/sirupsen/logrus"

	// Kubernetes Types

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createPod accepts Syringe-specific constructs like Endpoints and network definitions, and translates them
// into a Kubernetes pod object, and attempts to create it.
func (s *AntidoteScheduler) createPod(ep *models.LiveEndpoint, networks []string, req services.LessonScheduleRequest) (*corev1.Pod, error) {

	nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

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

	volumes, volumeMounts, initContainers := s.getVolumesConfiguration(req.LessonSlug)

	privileged := false

	// If the endpoint is a jupyter server, we don't want to append a curriculum version,
	// because that's part of the platform. For all others, we will append the version of the curriculum.
	var imageRef string
	if strings.Contains(ep.Image, "jupyter") {
		imageRef = ep.Image
	} else {

		image, err := s.Db.GetImage(ep.Image)
		if err != nil {
			return nil, fmt.Errorf("Unable to find referenced image %s in data store: %v", ep.Image, err)
		}
		privileged = image.Privileged
		imageRef = fmt.Sprintf("%s/%s:%s", s.Config.ImageOrg, ep.Image, s.Config.CurriculumVersion)
	}

	// TODO(mierdin): Here, you will want to do two things. Append the image org from the config.
	// Also, you will want to verify that the referenced image is loaded in the DB. This is because
	// we'll use metadata from it.

	pullPolicy := v1.PullAlways
	if s.Config.AlwaysPull {
		pullPolicy = v1.PullIfNotPresent
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ep.Name,
			Namespace: nsName,
			Labels: map[string]string{
				"liveLesson":     fmt.Sprintf("%d", req.LiveLessonID),
				"liveSession":    fmt.Sprintf("%d", req.LiveSessionID),
				"podName":        ep.Name,
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": string(netAnnotationsJSON),
			},
		},
		Spec: corev1.PodSpec{

			// All syringe-created pods are assigned to the same host for a given namespace. This keeps things much simplier, since each
			// network just uses linux bridges local to that host. Multi-host networking is a bit hit-or-miss when used with multus, so
			// this just keeps things simpler.
			// https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
			Affinity: &corev1.Affinity{
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"liveLesson":     fmt.Sprintf("%d", req.LiveLessonID),
									"liveSession":    fmt.Sprintf("%d", req.LiveSessionID),
									"syringeManaged": "yes",
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

	// TODO(mierdin): See Antidote mini-project 6 (MP6) for details on how we're planning to obviate
	// the need for privileged mode entirely. For now, this mechanism allows us to only grant this to
	// images that contain a virtualization layer (i.e. network devices).
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
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
			"networks":  string(netAnnotationsJSON),
		}).Infof("Created pod: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Pod %s already exists.", ep.Name)

		result, err := s.Client.CoreV1().Pods(nsName).Get(ep.Name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve pod after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating pod %s: %s", ep.Name, err)
		return nil, err
	}
	return result, err
}
