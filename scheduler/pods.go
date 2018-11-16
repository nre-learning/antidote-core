package scheduler

import (
	"encoding/json"
	"fmt"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	log "github.com/sirupsen/logrus"
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

func (ls *LessonScheduler) createPod(ep Endpoint, etype pb.LiveEndpoint_EndpointType, networks []string, req *LessonScheduleRequest) (*corev1.Pod, error) {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	nsName := fmt.Sprintf("%d-%s-ns", req.LessonDef.LessonId, req.Session)

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
			Name:      ep.GetName(),
			Namespace: nsName,
			Labels: map[string]string{
				"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonId),
				"sessionId":      req.Session,
				"endpointType":   etype.String(),
				"podName":        ep.GetName(),
				"syringeManaged": "yes",
				"sshUser":        ep.GetSshuser(),
				"sshPassword":    ep.GetSshpassword(),
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": string(netAnnotationsJson),
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
									"lessonId":       fmt.Sprintf("%d", req.LessonDef.LessonId),
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
					Name:  ep.GetName(),
					Image: ep.GetImage(),

					// Omitting in order to keep things speedy. For debugging, uncomment this, and the image will be pulled every time.
					ImagePullPolicy: "Always",

					// ImagePullPolicy: "IfNotPresent",

					Ports: []corev1.ContainerPort{}, // Will set below
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "git-volume",
							ReadOnly:  false,
							MountPath: ls.SyringeConfig.LessonRepoDir,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{
								"NET_ADMIN",
							},
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

	ports := ep.GetPorts()

	if etype.String() == "DEVICE" || etype.String() == "UTILITY" {
		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged:               &b,
			AllowPrivilegeEscalation: &b,
		}

		// Remove any existing port 22
		newports := []int32{}
		for p := range ports {
			if ports[p] != 22 {
				newports = append(newports, ports[p])
			}
		}

		// Add back in at the beginning, and append the rest.
		ports = append([]int32{22}, newports...)
	}

	// else if etype.String() == "IFRAME" {
	// 	port := req.LessonDef.Stages[req.Stage].IframeResource.Port
	// 	pod.Spec.Containers[0].Ports = append(pod.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: port})
	// }

	// Add any remaining ports not specified by the user
	for p := range ports {
		pod.Spec.Containers[0].Ports = append(pod.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: ports[p]})
	}

	result, err := coreclient.Pods(nsName).Create(pod)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
			"networks":  string(netAnnotationsJson),
		}).Infof("Created pod: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Pod %s already exists.", ep.GetName())

		result, err := coreclient.Pods(nsName).Get(ep.GetName(), metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve pod after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating pod %s: %s", ep.GetName(), err)
		return nil, err
	}
	return result, err
}
