package scheduler

import (
	"fmt"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	log "github.com/sirupsen/logrus"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	betav1client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
)

func (ls *LessonScheduler) createIngress(nsName string, ifr *pb.IframeResource) (*v1beta1.Ingress, error) {

	betaclient, err := betav1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	newIngress := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ifr.Ref,
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				"ingress.kubernetes.io/ingress.class":           "nginx",
				"ingress.kubernetes.io/ssl-services":            ifr.Ref,
				"ingress.kubernetes.io/ssl-redirect":            "true",
				"ingress.kubernetes.io/force-ssl-redirect":      "true",
				"ingress.kubernetes.io/rewrite-target":          "/",
				"nginx.ingress.kubernetes.io/rewrite-target":    "/",
				"nginx.ingress.kubernetes.io/limit-connections": "10",
				"nginx.ingress.kubernetes.io/limit-rps":         "5",
			},
		},

		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts:      []string{ls.SyringeConfig.Domain},
					SecretName: "tls-certificate",
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: ls.SyringeConfig.Domain,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: fmt.Sprintf("/%s-%s", nsName, ifr.Ref),
									Backend: v1beta1.IngressBackend{
										ServiceName: ifr.Ref,
										ServicePort: intstr.FromInt(int(ifr.Port)),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	result, err := betaclient.Ingresses(nsName).Create(&newIngress)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created ingress: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Ingress %s already exists.", ifr.Ref)

		result, err := betaclient.Ingresses(nsName).Get(ifr.Ref, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve ingress after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating ingress %s: %s", ifr.Ref, err)
		return nil, err
	}

	return result, nil

}
