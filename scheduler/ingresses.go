package scheduler

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	// Kubernetes types
	v1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (ls *LessonScheduler) createIngress(nsName string, ep *pb.Endpoint, port int32) (*v1beta1.Ingress, error) {

	redir := "true"

	// temporary but functional hack to disable SSL redirection for selfmedicate
	// (doesn't currently use HTTPS)
	if ls.SyringeConfig.Domain == "antidote-local" {
		redir = "false"
	}

	newIngress := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ep.Name,
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				"ingress.kubernetes.io/ingress.class":      "nginx",
				"ingress.kubernetes.io/ssl-services":       ep.Name,
				"ingress.kubernetes.io/ssl-redirect":       redir,
				"ingress.kubernetes.io/force-ssl-redirect": redir,
				// "ingress.kubernetes.io/rewrite-target":          "/",
				// "nginx.ingress.kubernetes.io/rewrite-target":    "/",
				"nginx.ingress.kubernetes.io/limit-connections": "10",
				"nginx.ingress.kubernetes.io/limit-rps":         "5",
				"nginx.ingress.kubernetes.io/add-base-url":      "true",
				// "nginx.ingress.kubernetes.io/app-root":          "/13-jjtigg867ghr3gye-ns-jupyter/",
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

					// TODO(mierdin): need to build this based on incoming protocol from syringefile. Might need to be HTTPS
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: fmt.Sprintf("/%s-%s", nsName, ep.Name),
									Backend: v1beta1.IngressBackend{
										ServiceName: ep.Name,
										ServicePort: intstr.FromInt(int(port)),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	result, err := ls.Client.ExtensionsV1beta1().Ingresses(nsName).Create(&newIngress)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created ingress: %s", result.ObjectMeta.Name)

	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Ingress %s already exists.", ep.Name)

		result, err := ls.Client.ExtensionsV1beta1().Ingresses(nsName).Get(ep.Name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve ingress after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating ingress %s: %s", ep.Name, err)
		return nil, err
	}

	return result, nil

}
