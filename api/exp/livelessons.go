package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	models "github.com/nre-learning/syringe/db/models"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

func (s *SyringeAPIServer) RequestLiveLesson(ctx context.Context, lp *pb.LessonParams) (*pb.LessonUUID, error) {

	// TODO(mierdin) look up session ID in DB first
	if lp.SessionId == "" {
		msg := "Session ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	if lp.LessonId == 0 {
		msg := "Lesson ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	if lp.LessonStage == 0 {
		msg := "Stage ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	// Identify lesson definition - return error if doesn't exist by ID
	_, err := s.Db.GetLesson(lp.LessonId)
	if err != nil {
		log.Errorf("Couldn't find lesson ID %d", lp.LessonId)
		return &pb.LessonUUID{}, errors.New("Failed to find referenced lesson ID")
	}

	// Ensure requested stage is present. We add a zero-index stage on import to each lesson so that
	// stage ID 1 refers to the second index (1) in the stage slice.
	// So, to check that the requested stage exists, the length of the slice must be equal or greater than the
	// requested stage + 1. I.e. if there's only one stage, the slice will have a length of 2
	// if len(s.Scheduler.Curriculum.Lessons[lp.LessonId].Stages) < int(lp.LessonStage) {
	// 	msg := "Invalid stage ID for this lesson"
	// 	log.Error(msg)
	// 	return nil, errors.New(msg)
	// }

	// Check to see if the livelesson already exists in an errored state.
	// If so, return error. TODO(mierdin): Consider this further, maybe we can do something to clean up the
	// LiveLesson and the underlying resources and try again?
	llID := fmt.Sprintf("%d-%s", lp.LessonId, lp.SessionId)
	ll, err := s.Db.GetLiveLesson(llID)
	// LiveLesson already exists, so we should handle this accordingly
	if err == nil {
		if ll.Error {
			return &pb.LessonUUID{}, errors.New("Lesson is in errored state. Please try again later")
		}

		// TODO(mierdin): Is this the right place for this?
		if ll.Busy {
			return &pb.LessonUUID{}, errors.New("LiveLesson is currently busy")
		}

		// If the incoming requested LessonStage is different from the current livelesson state,
		// tell the scheduler to change the state
		if ll.LessonStage != lp.LessonStage {

			// Update state
			ll.LessonStage = lp.LessonStage
			ll.Busy = true
			s.Db.UpdateLiveLesson(ll)

			// Request the schedule move forward with stage change activities
			req := &scheduler.LessonScheduleRequest{
				Operation:    scheduler.OperationType_MODIFY,
				Stage:        lp.LessonStage,
				LiveLessonID: llID,
			}
			s.Requests <- req

		} else {

			// Nothing to do but the user did interact with this lesson so we should boop it.
			req := &scheduler.LessonScheduleRequest{
				Operation:    scheduler.OperationType_BOOP,
				LiveLessonID: llID,
			}
			s.Requests <- req
		}

		return &pb.LessonUUID{Id: llID}, nil
	}

	// If we get here, the LiveLesson doesn't exist, and we should continue with a workflow for creating one

	// 3 - if doesn't already exist, put together schedule request and send to channel

	// Initialize new LiveLesson
	s.Db.CreateLiveLesson(&models.LiveLesson{
		ID:            llID,
		SessionID:     lp.SessionId,
		LessonID:      lp.LessonId,
		LiveEndpoints: map[string]*models.LiveEndpoint{},
		LessonStage:   lp.LessonStage,
		Busy:          true,
		Status:        "INITIAL_BOOT",
		CreatedTime:   time.Now(),
	})

	// Pre-emptively populate livelessons map with initial status.
	// This will be updated when the Scheduler response comes back.
	s.SetLiveLesson(lessonUuid, &pb.LiveLesson{
		LiveLessonStatus: pb.Status_INITIAL_BOOT,
		LessonId:         lp.LessonId,
		LessonUUID:       lessonUuid,
		LessonStage:      lp.LessonStage,
	})

	req := &scheduler.LessonScheduleRequest{
		Operation:    scheduler.OperationType_CREATE,
		Stage:        lp.LessonStage,
		LiveLessonID: llID,
		Created:      time.Now(),
	}
	s.Requests <- req

	return &pb.LessonUUID{Id: llID}, nil
}

func (s *SyringeAPIServer) GetSyringeState(ctx context.Context, _ *empty.Empty) (*pb.SyringeState, error) {
	return &pb.SyringeState{
		Livelessons: s.LiveLessonState,
	}, nil
}

func (s *SyringeAPIServer) HealthCheck(ctx context.Context, _ *empty.Empty) (*pb.HealthCheckMessage, error) {
	return &pb.HealthCheckMessage{}, nil
}

func (s *SyringeAPIServer) GetLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.LiveLesson, error) {

	if uuid.Id == "" {
		msg := "Lesson UUID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	if !s.LiveLessonExists(uuid.Id) {
		return nil, errors.New("livelesson not found")
	}

	ll := s.LiveLessonState[uuid.Id]

	if ll.Error {
		return nil, errors.New("Livelesson encountered errors during provisioning. See syringe logs")
	}
	return ll, nil

}

func (s *SyringeAPIServer) AddSessiontoGCWhitelist(ctx context.Context, session *pb.Session) (*pb.HealthCheckMessage, error) {
	s.Scheduler.GcWhiteListMu.Lock()
	defer s.Scheduler.GcWhiteListMu.Unlock()

	if _, ok := s.Scheduler.GcWhiteList[session.Id]; ok {
		return nil, fmt.Errorf("session %s already present in whitelist", session.Id)
	}

	s.Scheduler.GcWhiteList[session.Id] = session

	return nil, nil
}

func (s *SyringeAPIServer) RemoveSessionFromGCWhitelist(ctx context.Context, session *pb.Session) (*pb.HealthCheckMessage, error) {
	s.Scheduler.GcWhiteListMu.Lock()
	defer s.Scheduler.GcWhiteListMu.Unlock()

	if _, ok := s.Scheduler.GcWhiteList[session.Id]; !ok {
		return nil, fmt.Errorf("session %s not found in whitelist", session.Id)
	}

	delete(s.Scheduler.GcWhiteList, session.Id)

	return nil, nil

}

func (s *SyringeAPIServer) GetGCWhitelist(ctx context.Context, _ *empty.Empty) (*pb.Sessions, error) {
	sessions := []*pb.Session{}

	for id := range s.Scheduler.GcWhiteList {
		sessions = append(sessions, &pb.Session{Id: id})
	}

	return &pb.Sessions{
		Sessions: sessions,
	}, nil
}

func (s *SyringeAPIServer) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessons, error) {
	return &pb.LiveLessons{Items: s.LiveLessonState}, nil
}

func (s *SyringeAPIServer) KillLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.KillLiveLessonStatus, error) {

	if _, ok := s.LiveLessonState[uuid.Id]; !ok {
		return nil, errors.New("Livelesson not found")
	}

	s.Scheduler.Requests <- &scheduler.LessonScheduleRequest{
		Operation: scheduler.OperationType_DELETE,
		Uuid:      uuid.Id,
	}

	return &pb.KillLiveLessonStatus{Success: true}, nil
}

func (s *SyringeAPIServer) RequestVerification(ctx context.Context, uuid *pb.LessonUUID) (*pb.VerificationTaskUUID, error) {

	if _, ok := s.LiveLessonState[uuid.Id]; !ok {
		return nil, errors.New("Livelesson not found")
	}
	ll := s.LiveLessonState[uuid.Id]

	if ld, ok := s.Scheduler.Curriculum.Lessons[ll.LessonId]; !ok {
		// Unlikely to happen since we've verified the livelesson exists,
		// but easy to check
		return nil, errors.New("Invalid lesson ID")
	} else {
		if !ld.Stages[ll.LessonStage].VerifyCompleteness {
			return nil, errors.New("This lesson's stage doesn't include a completeness verification check")
		}
	}

	vtUUID := fmt.Sprintf("%s-%d", uuid.Id, ll.LessonStage)

	// If it already exists we can return it right away
	if _, ok := s.VerificationTasks[vtUUID]; ok {
		return &pb.VerificationTaskUUID{Id: vtUUID}, nil
	}

	// Proceed with the creation of a new verification task
	newVt := &pb.VerificationTask{
		LiveLessonId:    ll.LessonUUID,
		LiveLessonStage: ll.LessonStage,
		Working:         true,
		Success:         false,
		Message:         "Starting verification",
	}
	s.SetVerificationTask(vtUUID, newVt)

	s.Scheduler.Requests <- &scheduler.LessonScheduleRequest{
		Lesson:    s.Scheduler.Curriculum.Lessons[ll.LessonId],
		Operation: scheduler.OperationType_VERIFY,
		Stage:     ll.LessonStage,
		Uuid:      uuid.Id,
		Created:   time.Now(),
	}

	return &pb.VerificationTaskUUID{Id: vtUUID}, nil
}

func (s *SyringeAPIServer) GetVerification(ctx context.Context, vtUUID *pb.VerificationTaskUUID) (*pb.VerificationTask, error) {
	if _, ok := s.VerificationTasks[vtUUID.Id]; !ok {
		return nil, errors.New("verification task UUID not found")
	}
	return s.VerificationTasks[vtUUID.Id], nil
}
