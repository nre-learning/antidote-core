package scheduler

import (
	"fmt"

	ext "github.com/opentracing/opentracing-go/ext"

	ot "github.com/opentracing/opentracing-go"
	log "github.com/opentracing/opentracing-go/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// syncSecret takes care of copying secrets (e.g. TLS certificates, Docker pull
// credentials, etc) from a configured location into the lesson namespace.
// Kubernetes does not allow cross-namespace secret lookups, so this allows us to store
// secrets in a non-volatile namespace and then copy them into the volatile lesson namespace
// at runtime.
func (s *AntidoteScheduler) syncSecret(sc ot.SpanContext, sourceNs, destNs, secretName string) error {
	span := ot.StartSpan("scheduler_secret_sync", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("sourceNs", sourceNs)
	span.SetTag("destNs", destNs)
	span.SetTag("secretName", secretName)

	sourceSecret, err := s.Client.CoreV1().Secrets(sourceNs).Get(secretName, metav1.GetOptions{})
	if err != nil {
		span.LogFields(
			log.String("message", "Failed to retrieve secret"),
			log.Error(err),
		)
		ext.Error.Set(span, true)
		return err
	}

	result, err := s.Client.CoreV1().Secrets(destNs).Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceSecret.ObjectMeta.Name,
			Namespace: destNs,
		},
		Data:       sourceSecret.Data,
		StringData: sourceSecret.StringData,
		Type:       sourceSecret.Type,
	})
	if err != nil {
		span.LogFields(
			log.String("message", fmt.Sprintf("Problem creating secret %s: %s", secretName, err)),
			log.Error(err),
		)
		ext.Error.Set(span, true)
		return err
	}
	span.LogEvent(fmt.Sprintf("Successfully copied secret %s", result.ObjectMeta.Name))
	return nil
}
