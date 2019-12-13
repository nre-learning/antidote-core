package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

func (s *SyringeAPIServer) handleResultCREATE(result *scheduler.LessonScheduleResult) {
	if !result.Success {
		log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
		s.SetLiveLesson(result.Uuid, &pb.LiveLesson{Error: true})
		return
	}

	s.SetLiveLesson(result.Uuid, s.Scheduler.KubeLabs[result.Uuid].ToLiveLesson())
}

func (s *SyringeAPIServer) handleResultMODIFY(result *scheduler.LessonScheduleResult) {
	if !result.Success {
		log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
		s.SetLiveLesson(result.Uuid, &pb.LiveLesson{Error: true})
		return
	}
	ll := s.Scheduler.KubeLabs[result.Uuid].ToLiveLesson()
	s.SetLiveLesson(result.Uuid, ll)
}

func (s *SyringeAPIServer) handleResultVERIFY(result *scheduler.LessonScheduleResult) {
	vtUUID := fmt.Sprintf("%s-%d", result.Uuid, result.Stage)

	vt := s.VerificationTasks[vtUUID]
	vt.Working = false
	vt.Success = result.Success
	if result.Success == true {
		vt.Message = "Successfully verified"
	} else {

		// TODO(mierdin): Provide an optional field for the author to provide a hint that overrides this.
		vt.Message = "Failed to verify"
	}
	vt.Completed = &timestamp.Timestamp{
		Seconds: time.Now().Unix(),
	}

	s.SetVerificationTask(vtUUID, vt)

}

// handleResultDELETE runs in response to a scheduler deletion event by removing any tracked state at the API layer.
func (s *SyringeAPIServer) handleResultDELETE(result *scheduler.LessonScheduleResult) {
	uuid := strings.TrimSuffix(result.Uuid, "-ns")
	s.DeleteLiveLesson(uuid)
}

func (s *SyringeAPIServer) handleResultBOOP(result *scheduler.LessonScheduleResult) {}
