package scheduler

import (
	"errors"
	"fmt"
	"strings"

	ext "github.com/opentracing/opentracing-go/ext"

	ot "github.com/opentracing/opentracing-go"
	log "github.com/opentracing/opentracing-go/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// syncCertificate takes care of copying the primary TLS certificate from a configured
// location into the lesson namespace. This is required because Kubernetes does not
// allow cross-namespace secret lookups, and we need to be able to offer TLS for
// http presentation endpoints.
func (s *AntidoteScheduler) syncCertificate(sc ot.SpanContext, nsName string) error {
	span := ot.StartSpan("scheduler_secret_certificate_sync", ot.ChildOf(sc))
	defer span.Finish()

	// Determine location of original certificate based from config
	var certNs = "prod"
	var certName = "tls-certificate"
	certLocations := strings.Split(s.Config.CertLocation, "/")
	if len(certLocations) == 2 {
		certNs = certLocations[0]
		certName = certLocations[1]
	}

	prodCert, err := s.Client.CoreV1().Secrets(certNs).Get(certName, metav1.GetOptions{})
	if err != nil {
		span.LogFields(
			log.String("message", "Failed to retrieve secret"),
			log.Error(err),
		)
		ext.Error.Set(span, true)
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

	result, err := s.Client.CoreV1().Secrets(nsName).Create(&newCert)
	if err == nil {
		span.LogEvent(fmt.Sprintf("Successfully copied secret %s", result.ObjectMeta.Name))
	} else if apierrors.IsAlreadyExists(err) {
		span.LogEvent(fmt.Sprintf("Secret %s already exists.", newCert.ObjectMeta.Name))
		return nil
	} else {
		span.LogFields(
			log.String("message", fmt.Sprintf("Problem creating secret %s: %s", newCert.ObjectMeta.Name, err)),
			log.Error(err),
		)
		ext.Error.Set(span, true)
		return err
	}
	return nil
}

// syncPullCreds takes care of copying pull credentials into the lesson namespace, per
// https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
func (s *AntidoteScheduler) syncPullCreds(sc ot.SpanContext, nsName string) error {
	span := ot.StartSpan("scheduler_secret_pullcreds_sync", ot.ChildOf(sc))
	defer span.Finish()

	if s.Config.PullCredsLocation == "" {
		span.LogEvent("No PullCredsLocation specified in the config, skipping.")
		return nil
	}

	locs := strings.Split(s.Config.PullCredsLocation, "/")
	if len(locs) != 2 {
		err := errors.New("Improper configuration for pull creds")
		span.LogFields(
			log.String("PullCredsLocation", s.Config.PullCredsLocation),
			log.Error(err),
		)
		ext.Error.Set(span, true)
		return err
	}

	pullSecret, err := s.Client.CoreV1().Secrets(locs[0]).Get(locs[1], metav1.GetOptions{})
	if err != nil {
		span.LogFields(
			log.String("message", "Failed to retrieve secret"),
			log.Error(err),
		)
		ext.Error.Set(span, true)
		return err
	}

	//Copy secret into this namespace
	pullSecret.ObjectMeta.Namespace = nsName

	newCert := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecret.ObjectMeta.Name,
			Namespace: pullSecret.ObjectMeta.Namespace,
		},
		Data:       pullSecret.Data,
		StringData: pullSecret.StringData,
		Type:       pullSecret.Type,
	}

	result, err := s.Client.CoreV1().Secrets(nsName).Create(&newCert)
	if err != nil {
		span.LogFields(
			log.String("message", fmt.Sprintf("Problem creating secret %s: %s", newCert.ObjectMeta.Name, err)),
			log.Error(err),
		)
		ext.Error.Set(span, true)
		return err
	}
	span.LogEvent(fmt.Sprintf("Successfully copied secret %s", result.ObjectMeta.Name))
	return nil
}
