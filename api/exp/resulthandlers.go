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

func (s *server) handleResponseCREATE(result *scheduler.LessonScheduleResult) {
	if !result.Success {
		log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
		s.SetLiveLesson(result.Uuid, &pb.LiveLesson{Error: true})
		return
	}

	s.recordProvisioningTime(result.ProvisioningTime, result)
	s.SetLiveLesson(result.Uuid, s.scheduler.KubeLabs[result.Uuid].ToLiveLesson())
}

func (s *server) handleResponseMODIFY(result *scheduler.LessonScheduleResult) {
	if !result.Success {
		log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
		s.SetLiveLesson(result.Uuid, &pb.LiveLesson{Error: true})
		return
	}
	s.SetLiveLesson(result.Uuid, s.scheduler.KubeLabs[result.Uuid].ToLiveLesson())
}

func (s *server) handleResponseVERIFY(result *scheduler.LessonScheduleResult) {
	vtUUID := fmt.Sprintf("%s-%d", result.Uuid, result.Stage)

	vt := s.verificationTasks[vtUUID]
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

func (s *server) handleResponseDELETE(result *scheduler.LessonScheduleResult) {
	// This is a bit awkward. Maybe change UUID to a slice? Then just act on all?
	if len(result.GCLessons) > 0 {
		for i := range result.GCLessons {
			uuid := strings.TrimRight(result.GCLessons[i], "-ns")
			s.DeleteLiveLesson(uuid)
		}
	} else {
		s.DeleteLiveLesson(result.Uuid)
	}
}

func (s *server) handleResponseBOOP(result *scheduler.LessonScheduleResult) {}
