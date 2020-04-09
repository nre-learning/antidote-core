package scheduler

import (
	"testing"

	ot "github.com/opentracing/opentracing-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyncSecret(t *testing.T) {
	span := ot.StartSpan("")
	defer span.Finish()

	s := createFakeScheduler()
	s.Config.SecretsNamespace = "prod"
	s.Config.PullCredName = "docker-pull-creds"

	_, err := s.Client.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Config.SecretsNamespace,
			Labels: map[string]string{
				"name": s.Config.SecretsNamespace,
			},
		},
	})
	ok(t, err)
	_, err = s.Client.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testns",
			Labels: map[string]string{
				"name": "testns",
			},
		},
	})
	ok(t, err)

	_, err = s.Client.CoreV1().Secrets("prod").Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Config.PullCredName,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": {1, 2, 3},
		},
	})
	ok(t, err)

	err = s.syncSecret(span.Context(), s.Config.SecretsNamespace, "testns", s.Config.PullCredName)
	ok(t, err)

	syncedSecret, err := s.Client.CoreV1().Secrets("testns").Get(s.Config.PullCredName, metav1.GetOptions{})
	ok(t, err)

	assert(t, syncedSecret.Type == "kubernetes.io/dockerconfigjson", "")
}
