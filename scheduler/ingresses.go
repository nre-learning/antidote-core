package scheduler

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	models "github.com/nre-learning/syringe/db/models"

	// Kubernetes types
	v1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (ls *LessonScheduler) createIngress(nsName string, ep *models.LiveEndpoint, p *models.LivePresentation) (*v1beta1.Ingress, error) {

	redir := "true"

	// temporary but functional hack to disable SSL redirection for selfmedicate
	// (doesn't currently use HTTPS)
	if ls.SyringeConfig.Domain == "antidote-local" || ls.SyringeConfig.Domain == "localhost" {
		redir = "false"
	}

	ingressDomain := fmt.Sprintf("%s-%s-%s.heps.%s", nsName, ep.Name, p.Name, ls.SyringeConfig.Domain)

	newIngress := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", ep.Name, p.Name),
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
			},
			Annotations: map[string]string{
				"ingress.kubernetes.io/ingress.class": "nginx",

				// https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/ssl-services
				// We only need this if the endpoint requires HTTPS termination.
				"ingress.kubernetes.io/ssl-services": ep.Name,

				"ingress.kubernetes.io/ssl-redirect":       redir,
				"ingress.kubernetes.io/force-ssl-redirect": redir,
				// "ingress.kubernetes.io/rewrite-target":          "/",
				// "nginx.ingress.kubernetes.io/rewrite-target":    "/",
				// "nginx.ingress.kubernetes.io/limit-connections": "10",
				// "nginx.ingress.kubernetes.io/limit-rps":         "5",
				// "nginx.ingress.kubernetes.io/add-base-url":      "true",
				// "nginx.ingress.kubernetes.io/app-root":          "/13-jjtigg867ghr3gye-ns-jupyter/",
			},
		},

		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts:      []string{ingressDomain},
					SecretName: "tls-certificate",
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: ingressDomain,

					// TODO(mierdin): need to build this based on incoming protocol from syringefile. Might need to be HTTPS
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: ep.Name,
										ServicePort: intstr.FromInt(int(p.Port)),
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
