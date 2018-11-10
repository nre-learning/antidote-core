package scheduler

import (
	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (ls *LessonScheduler) syncSecret(nsName string) error {

	coreclient, err := corev1client.NewForConfig(ls.KubeConfig)
	if err != nil {
		panic(err)
	}

	prodCert, err := coreclient.Secrets("prod").Get("tls-certificate", metav1.GetOptions{})
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

	result, err := coreclient.Secrets(nsName).Create(&newCert)
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
