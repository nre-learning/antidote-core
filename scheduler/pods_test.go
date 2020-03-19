package scheduler

import (
	"testing"

	models "github.com/nre-learning/antidote-core/db/models"
	services "github.com/nre-learning/antidote-core/services"
)

// TestPods is responsible for ensuring kubernetes pods are created as expected, with expected
// properties set based on Syringe-specific inputs.
func TestPods(t *testing.T) {

	// SETUP
	schedulerSvc := createFakeScheduler()

	// Test normal pod creation
	t.Run("A=1", func(t *testing.T) {

		pod, err := schedulerSvc.createPod(
			&models.LiveEndpoint{
				Name:  "linux1",
				Image: "utility",
				Presentations: []*models.LivePresentation{
					{Name: "cli", Type: "ssh", Port: 22},
				},
				Ports: []int32{22},
			},
			[]string{"1", "2", "3"},
			services.LessonScheduleRequest{
				LiveLessonID: "asdf",
			},
		)

		// Assert pod exists without error
		ok(t, err)
		assert(t, (pod != nil), "")

		// Assert created namespace is correct
		equals(t, pod.Namespace, "antidote-testing-asdf")

		// TODO(mierdin): Assert expected networks exist properly

	})

}
