package scheduler

import (
	"encoding/json"
	"testing"

	services "github.com/nre-learning/antidote-core/services"
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

	schedulerSvc := createFakeScheduler()

	t.Run("A=1", func(t *testing.T) {

		network, err := schedulerSvc.createNetwork(
			0,
			"vqfx1-vqfx2",
			services.LessonScheduleRequest{
				LiveLessonID: "asdf",
			},
		)
		ok(t, err)

		var nc CniNetconf
		err = json.Unmarshal([]byte(network.Spec.Config), &nc)
		ok(t, err)

		assert(t, nc.Ipam.Subnet == "10.10.0.0/16", "")
	})
}
