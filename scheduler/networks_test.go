package scheduler

import (
	"encoding/json"
	"testing"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	kubernetesCrdFake "github.com/nre-learning/syringe/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	kubernetesExtFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// TestNetworks is responsible for ensuring Syringe-imposed networking policies are working
func TestNetworks(t *testing.T) {

	type CniDelegate struct {
		HairpinMode bool `json:"hairpinMode,omitempty"`
	}

	type CniIpam struct {
		IpamType string `json:"type,omitempty"`
		Subnet   string `json:"subnet,omitempty"`
	}

	type CniNetconf struct {
		Name         string      `json:"name,omitempty"`
		Cnitype      string      `json:"type,omitempty"`
		Plugin       string      `json:"plugin,omitempty"`
		Bridge       string      `json:"bridge,omitempty"`
		ForceAddress bool        `json:"forceAddress,omitempty"`
		HairpinMode  bool        `json:"hairpinMode,omitempty"`
		Delegate     CniDelegate `json:"delegate,omitempty"`
		Ipam         CniIpam     `json:"ipam,omitempty"`
	}

	// SETUP
	nsName := "1-foobar-ns"
	syringeConfig := &config.SyringeConfig{
		CurriculumDir: "/antidote",
		Domain:        "localhost",
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: nsName,
		},
	}
	lessonScheduler := LessonScheduler{
		SyringeConfig: syringeConfig,
		Client:        testclient.NewSimpleClientset(namespace),
		ClientExt:     kubernetesExtFake.NewSimpleClientset(),
		ClientCrd:     kubernetesCrdFake.NewSimpleClientset(),
	}
	uuid := "1-abcdef"
	// END SETUP

	t.Run("A=1", func(t *testing.T) {

		network, err := lessonScheduler.createNetwork(
			0,
			"vqfx1-vqfx2",
			&LessonScheduleRequest{
				Uuid: uuid,
				Lesson: &pb.Lesson{
					LessonId: 1,
				},
			},
		)
		ok(t, err)

		var nc CniNetconf
		err = json.Unmarshal([]byte(network.Spec.Config), &nc)
		ok(t, err)

		assert(t, nc.Ipam.Subnet == "10.10.0.0/16", "")
	})
}
