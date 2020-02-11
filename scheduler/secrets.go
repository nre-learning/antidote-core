package scheduler

import (

	// Kubernetes types
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// syncSecret takes care of copying the primary TLS certificate from a configured
// location into the lesson namespace. This is required because Kubernetes does not
// allow cross-namespace secret lookups, and we need to be able to offer TLS for
// http presentation endpoints.
func (ls *LessonScheduler) syncSecret(nsName string) error {

	// Determine location of original certificate based from config
	var certNs = "prod"
	var certName = "tls-certificate"
	certLocations := strings.Split(ls.SyringeConfig.CertLocation, "/")
	if len(certLocations) == 2 {
		certNs = certLocations[0]
		certName = certLocations[1]
	}

	prodCert, err := ls.Client.CoreV1().Secrets(certNs).Get(certName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	//Copy secret into this namespace
	prodCert.ObjectMeta.Namespace = nsName

	newCert := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prodCert.ObjectMeta.Name,
			Namespace: prodCert.ObjectMeta.Namespace,
		},
		Data:       prodCert.Data,
		StringData: prodCert.StringData,
		Type:       prodCert.Type,
	}

	result, err := ls.Client.CoreV1().Secrets(nsName).Create(&newCert)
	if err == nil {
		log.WithFields(log.Fields{
			"namespace": nsName,
		}).Infof("Created secret: %s", result.ObjectMeta.Name)
	} else if apierrors.IsAlreadyExists(err) {
		log.Warnf("Secret %s already exists.", newCert.ObjectMeta.Name)
		return nil
	} else {
		log.Errorf("Problem creating secret %s: %s", newCert.ObjectMeta.Name, err)
		return err
	}
	return nil
}
