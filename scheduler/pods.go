package scheduler

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LabScheduler) deletePod(name string) error {
	return nil
}

type networkAnnotation struct {
	Name string `json:"name"`
}

func (ls *LabScheduler) createPod(podName, image string, etype pb.LabEndpoint_EndpointType, networks []string, req *LabScheduleRequest) (*corev1.Pod, error) {

	coreclient, err := corev1client.NewForConfig(ls.Config)
	if err != nil {
		panic(err)
	}
	// podName := fmt.Sprintf("%s-%s-%s-pod", req.LabDef.LabID, req.Session, req.LabDef.LabName)
	// netName := fmt.Sprintf("%s-%s-net", req.LabDef.LabID, req.Session)
	nsName := fmt.Sprintf("%d-%s-ns", req.LabDef.LabID, req.Session)

	b := true

	netAnnotations := []networkAnnotation{}
	for n := range networks {
		netAnnotations = append(netAnnotations, networkAnnotation{Name: networks[n]})
	}

	netAnnotationsJson, err := json.Marshal(netAnnotations)
	if err != nil {
		log.Error(err)
	}

	typePortMap := map[string]int32{
		"DEVICE":   22,
		"NOTEBOOK": 8888,
	}
	// typeFileMap := map[string]string{
	// 	"NOTEBOOK": "lesson.ipynb",
	// }

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: nsName,
			Labels: map[string]string{
				"labId":          fmt.Sprintf("%d", req.LabDef.LabID),
				"sessionId":      req.Session,
				"endpointType":   etype.String(),
				"podName":        podName,
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": string(netAnnotationsJson),
			},
		},
		Spec: corev1.PodSpec{

			// Currently tying the lab pods to the same host since I'm running into a temporary issue with multi-host weave networks
			// https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
			Affinity: &corev1.Affinity{
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"labId":          fmt.Sprintf("%d", req.LabDef.LabID),
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
			Containers: []corev1.Container{
				{
					Name:  podName,
					Image: image,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: typePortMap[etype.String()],
						},
					},
				},
			},
		},
	}

	// TODO(mierdin): Need to get this from env
	lessonsPath := "/home/mierdin/antidote/lessons"

	if etype.String() == "DEVICE" {
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  "CSRX_ROOT_PASSWORD",
				Value: "Password1!",
			},
		}

		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged:               &b,
			AllowPrivilegeEscalation: &b,
		}
	}

	if etype.String() == "NOTEBOOK" {
		pod.Spec.Volumes = []corev1.Volume{
			{
				Name: "notebook",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: fmt.Sprintf("%s/lesson-%s/lesson.ipynb", lessonsPath, strconv.Itoa(int(req.LabDef.LabID))),
					},
				},
			},
		}
		pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "notebook",
				ReadOnly:  true,
				MountPath: "/home/jovyan/work/lesson.ipynb",
			},
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

// ---
// apiVersion: v1
// kind: Pod
// metadata:
//   name: csrx1
//   # namespace: lesson1-abcdef
//   labels:
//     antidote_lab: "1"
//     lab_instance: "1"
//     podname: "csrx1"
//   annotations:
//     networks: '[
//         { "name": "weave-lab1" },
//         { "name": "weave-lab1" },
//         { "name": "weave-lab1" }
//     ]'
// spec:

//   # Currently tying the lab pods to the same host since I'm running into a temporary issue with multi-host weave networks
//   # https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
//   affinity:
//       podAffinity:
//         requiredDuringSchedulingIgnoredDuringExecution:
//         - labelSelector:
//             matchExpressions:
//             - key: antidote_lab
//               operator: In
//               values:
//               - "1"
//           topologyKey: kubernetes.io/hostname
//   containers:
//   - name: csrx1
//     image:
//     securityContext:
//       privileged: true
//       allowPrivilegeEscalation: true
//     env:
//     - name: CSRX_ROOT_PASSWORD
//       value: Password1!
//     ports:
//     - containerPort: 22
//     - containerPort: 830
