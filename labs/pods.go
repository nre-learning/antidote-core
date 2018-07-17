package labs

import (
	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LabScheduler) deletePod(name string) error {
	// // Create a new clientset which include our CRD schema
	// crdcs, scheme, err := crd.NewClient(ls.Config)
	// if err != nil {
	// 	panic(err)
	// }

	// // Create a CRD client interface
	// crdclient := client.CrdClient(crdcs, scheme, "default")

	// err = crdclient.Delete(name, &meta_v1.DeleteOptions{})
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (ls *LabScheduler) createPod(name, labId, labInstanceId string) (*corev1.Pod, error) {

	coreclient, err := corev1client.NewForConfig(ls.Config)
	if err != nil {
		panic(err)
	}
	podName := labId + labInstanceId + "pod" + name
	b := true

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"labId":          labId,
				"labInstanceId":  labInstanceId,
				"podName":        podName,
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				// TODO Obviously need to make this more dynamic
				"networks": "[ { 'name': 'lab001lab001-0001net' }, { 'name': 'lab001lab001-0001net' }, { 'name': 'lab001lab001-0001net' } ]",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: "csrx:18.1R1.9",
					SecurityContext: &corev1.SecurityContext{
						Privileged:               &b,
						AllowPrivilegeEscalation: &b,
					},
				},
			},
		},
	}

	result, err := coreclient.Pods("default").Create(pod)
	if err == nil {
		log.Infof("Created pod: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Pod %s already exists.", podName)

		// In this case we are returning what we tried to create. This means that when this lab is cleaned up,
		// syringe will delete the pod that already existed.
		return pod, err
	} else {
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
