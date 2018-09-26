package scheduler

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LessonScheduler) deletePod(name string) error {
	return nil
}

type networkAnnotation struct {
	Name string `json:"name"`
}

func (ls *LessonScheduler) createPod(podName, image string, etype pb.Endpoint_EndpointType, networks []string, req *LessonScheduleRequest) (*corev1.Pod, error) {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	nsName := fmt.Sprintf("%d-%s-ns", req.LessonDef.LessonID, req.Session)

	b := true

	netAnnotations := []networkAnnotation{}
	for n := range networks {
		netAnnotations = append(netAnnotations, networkAnnotation{Name: networks[n]})
	}

	netAnnotationsJson, err := json.Marshal(netAnnotations)
	if err != nil {
		log.Error(err)
	}

	defaultGitFileMode := int32(0755)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonID),
				"sessionId":      req.Session,
				"endpointType":   etype.String(),
				"podName":        podName,
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": string(netAnnotationsJson),

				// k8s.v1.cni.cncf.io/networks: '[
				// 	{ "name": "12-net" },
				// 	{ "name": "23-net" }
				// ]'
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
									"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonID),
									"sessionId":      req.Session,
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

			InitContainers: []corev1.Container{
				{
					Name:  "git-clone",
					Image: "alpine/git",
					Command: []string{
						"/usr/local/git/git-clone.sh",
					},
					Args: []string{
						"https://github.com/nre-learning/antidote.git",
						"master",
						"/antidote",
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
							MountPath: "/antidote",
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  podName,
					Image: image,

					ImagePullPolicy: "Always",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: typePortMap[etype.String()],
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "git-volume",
							ReadOnly:  false,
							MountPath: "/antidote",
						},
					},
				},
			},

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
	}

	if etype.String() == "DEVICE" {
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				//TODO(mierdin): need to change the image to not require this
				Name:  "VQFX_HOSTNAME",
				Value: podName,
			},
		}

		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged:               &b,
			AllowPrivilegeEscalation: &b,
		}
	}

	result, err := coreclient.Pods(nsName).Create(pod)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
			"networks":  string(netAnnotationsJson),
		}).Infof("Created pod: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Pod %s already exists.", podName)

		result, err := coreclient.Pods(nsName).Get(podName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve pod after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating pod %s: %s", podName, err)
		return nil, err
	}
	return result, err
}
