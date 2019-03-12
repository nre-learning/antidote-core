package api

import (
	"sync"
	"testing"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
)

// TestHandleResultDELETE verifies that the DELETE result handler cleans up
// livelesson state appropriately.
func TestHandleResultDELETE(t *testing.T) {
	apiServer := &server{
		liveLessonState: map[string]*pb.LiveLesson{
			"100-foobar": &pb.LiveLesson{},
			"200-foobar": &pb.LiveLesson{},
			"300-foobar": &pb.LiveLesson{},
		},
		liveLessonsMu: &sync.Mutex{},
	}

	apiServer.handleResultDELETE(&scheduler.LessonScheduleResult{
		Uuid: "200-foobar-ns",
	})

	assert(t, (len(apiServer.liveLessonState) == 2), "")
}
