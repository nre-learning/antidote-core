package scheduler

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	def "github.com/nre-learning/syringe/def"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	betav1client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
)

func (ls *LessonScheduler) createIngress(svcBuddy *corev1.Service, ifr *def.IframeResource) (*v1beta1.Ingress, error) {

	betaclient, err := betav1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	//TODO: Make this configurable
	domain := "ptr.labs.networkreliability.engineering"
	// var domain string
	// if ls.SyringeConfig.Tier == "ptr" {
	// 	domain = "ptr.labs.networkreliability.engineering"
	// } else {
	// 	domain = "labs.networkreliability.engineering"
	// }

	nsName := svcBuddy.ObjectMeta.Namespace

	newIngress := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ifr.Name,
			Namespace: nsName,
			Labels: map[string]string{
				"syringeManaged": "yes",
				"endpointType":   svcBuddy.ObjectMeta.Labels["endpointType"],
			},
			Annotations: map[string]string{
				"ingress.kubernetes.io/ingress.class":           "nginx",
				"ingress.kubernetes.io/ssl-services":            ifr.Name,
				"ingress.kubernetes.io/ssl-redirect":            "true",
				"ingress.kubernetes.io/force-ssl-redirect":      "true",
				"ingress.kubernetes.io/rewrite-target":          fmt.Sprintf("/%s-%s", nsName, ifr.Name),
				"nginx.ingress.kubernetes.io/rewrite-target":    fmt.Sprintf("/%s-%s", nsName, ifr.Name),
				"nginx.ingress.kubernetes.io/limit-connections": "10",
				"nginx.ingress.kubernetes.io/limit-rps":         "5",
			},
		},

		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{domain},

					//TODO(mierdin): Prudent to do a check for this
					SecretName: "tls-certificate",
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: domain,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: ifr.URI,
									Backend: v1beta1.IngressBackend{
										ServiceName: ifr.Name,
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
		log.Warnf("Ingress %s already exists.", ifr.Name)

		result, err := betaclient.Ingresses(nsName).Get(ifr.Name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Couldn't retrieve pod after failing to create a duplicate: %s", err)
			return nil, err
		}
		return result, nil
	} else {
		log.Errorf("Problem creating ingress %s: %s", ifr.Name, err)
		return nil, err
	}

	return result, nil

}
