package scheduler

import (
	"fmt"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	models "github.com/nre-learning/antidote-core/db/models"

	// Kubernetes types
	v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (s *AntidoteScheduler) createIngress(sc ot.SpanContext, nsName string, ep *models.LiveEndpoint, p *models.LivePresentation) (*v1beta1.Ingress, error) {
	span := ot.StartSpan("scheduler_ingress_create", ot.ChildOf(sc))
	span.SetTag("epName", ep.Name)
	span.SetTag("nsName", nsName)
	defer span.Finish()

	redir := "true"

	// temporary but functional hack to disable SSL redirection for selfmedicate
	// (doesn't currently use HTTPS)
	if s.Config.Domain == "antidote-local" || s.Config.Domain == "localhost" {
		redir = "false"
	}

	ingressDomain := fmt.Sprintf("%s-%s-%s.heps.%s", nsName, ep.Name, p.Name, s.Config.Domain)

	newIngress := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", ep.Name, p.Name),
			Namespace: nsName,
			Labels: map[string]string{
				"antidoteManaged": "yes",
			},
			Annotations: map[string]string{
				"ingress.kubernetes.io/ingress.class": "nginx",
				// https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/ssl-services
				// We only need this if the endpoint requires HTTPS termination.
				"ingress.kubernetes.io/ssl-services":       ep.Name,
				"ingress.kubernetes.io/ssl-redirect":       redir,
				"ingress.kubernetes.io/force-ssl-redirect": redir,

				// Strip X-Frame-Options headers from http endpoints (these would prevent iframes)
				"ingress.kubernetes.io/configuration-snippet": "proxy_hide_header X-Frame-Options;",
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
	result, err := s.Client.ExtensionsV1beta1().Ingresses(nsName).Create(&newIngress)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	return result, nil

}
