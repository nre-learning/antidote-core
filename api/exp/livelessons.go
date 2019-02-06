package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

func (s *server) RequestLiveLesson(ctx context.Context, lp *pb.LessonParams) (*pb.LessonUUID, error) {

	// TODO(mierdin): need to perform some basic security checks here. Need to check incoming IP address
	// and do some rate-limiting if possible. Alternatively you could perform this on the Ingress
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

	lessonUuid := fmt.Sprintf("%d-%s", lp.LessonId, lp.SessionId)

	// Identify lesson definition - return error if doesn't exist by ID
	if _, ok := s.scheduler.LessonDefs[lp.LessonId]; !ok {
		log.Errorf("Couldn't find lesson ID %d", lp.LessonId)
		return &pb.LessonUUID{}, errors.New("Failed to find referenced lesson ID")
	}

	// Ensure requested stage is present. We add a zero-index stage on import to each lesson so that
	// stage ID 1 refers to the second index (1) in the stage slice.
	// So, to check that the requested stage exists, the length of the slice must be equal or greater than the
	// requested stage + 1. I.e. if there's only one stage, the slice will have a length of 2
	if len(s.scheduler.LessonDefs[lp.LessonId].Stages) < int(lp.LessonStage+1) {
		msg := "Invalid stage ID for this lesson"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	// Check to see if it already exists in memory. If it does, don't send provision request.
	// Just look it up and send UUID
	if s.LiveLessonExists(lessonUuid) {

		if s.liveLessonState[lessonUuid].LessonStage != lp.LessonStage {

			// Update in-memory state
			s.UpdateLiveLessonStage(lessonUuid, lp.LessonStage)

			// Request the schedule move forward with stage change activities
			req := &scheduler.LessonScheduleRequest{
				LessonDef: s.scheduler.LessonDefs[lp.LessonId],
				Operation: scheduler.OperationType_MODIFY,
				Stage:     lp.LessonStage,
				Uuid:      lessonUuid,
			}

			s.scheduler.Requests <- req

			s.recordRequestTSDB(req)

		} else {

			// Nothing to do but the user did interact with this lesson so we should boop it.
			req := &scheduler.LessonScheduleRequest{
				LessonDef: s.scheduler.LessonDefs[lp.LessonId],
				Operation: scheduler.OperationType_BOOP,
				Stage:     0,
				Uuid:      lessonUuid,
			}

			s.scheduler.Requests <- req

			s.recordRequestTSDB(req)
		}

		return &pb.LessonUUID{Id: lessonUuid}, nil
	}

	// 3 - if doesn't already exist, put together schedule request and send to channel
	req := &scheduler.LessonScheduleRequest{
		LessonDef: s.scheduler.LessonDefs[lp.LessonId],
		Operation: scheduler.OperationType_CREATE,
		Stage:     lp.LessonStage,
		Uuid:      lessonUuid,
		Created:   time.Now(),
	}
	s.scheduler.Requests <- req

	s.recordRequestTSDB(req)

	// Pre-emptively populate livelessons map with initial status.
	// This will be updated when the scheduler response comes back.
	s.SetLiveLesson(lessonUuid, &pb.LiveLesson{
		LiveLessonStatus: pb.Status_INITIAL_BOOT,
		LessonId:         lp.LessonId,
		LessonUUID:       lessonUuid,
		LessonStage:      lp.LessonStage,
	})

	return &pb.LessonUUID{Id: lessonUuid}, nil
}

func (s *server) GetSyringeState(ctx context.Context, _ *empty.Empty) (*pb.SyringeState, error) {
	return &pb.SyringeState{
		Livelessons: s.liveLessonState,
	}, nil
}

func (s *server) HealthCheck(ctx context.Context, _ *empty.Empty) (*pb.HealthCheckMessage, error) {
	return &pb.HealthCheckMessage{}, nil
}

func (s *server) GetLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.LiveLesson, error) {

	if uuid.Id == "" {
		msg := "Lesson UUID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	if !s.LiveLessonExists(uuid.Id) {
		return nil, errors.New("livelesson not found")
	}

	ll := s.liveLessonState[uuid.Id]

	if ll.Error {
		return nil, errors.New("Livelesson encountered errors during provisioning. See syringe logs")
	}

	// Remove all blackbox entries
	newEndpoints := map[string]*pb.LiveEndpoint{}
	for name, e := range ll.LiveEndpoints {
		if e.Type != pb.LiveEndpoint_BLACKBOX {
			newEndpoints[name] = e
		}
	}
	ll.LiveEndpoints = newEndpoints

	return ll, nil

}

func (s *server) AddSessiontoGCWhitelist(ctx context.Context, session *pb.Session) (*pb.HealthCheckMessage, error) {
	s.scheduler.GcWhiteListMu.Lock()
	defer s.scheduler.GcWhiteListMu.Unlock()

	if _, ok := s.scheduler.GcWhiteList[session.Id]; ok {
		return nil, fmt.Errorf("session %s already present in whitelist", session.Id)
	}

	s.scheduler.GcWhiteList[session.Id] = session

	return nil, nil
}

func (s *server) RemoveSessionFromGCWhitelist(ctx context.Context, session *pb.Session) (*pb.HealthCheckMessage, error) {
	s.scheduler.GcWhiteListMu.Lock()
	defer s.scheduler.GcWhiteListMu.Unlock()

	if _, ok := s.scheduler.GcWhiteList[session.Id]; !ok {
		return nil, fmt.Errorf("session %s not found in whitelist", session.Id)
	}

	delete(s.scheduler.GcWhiteList, session.Id)

	return nil, nil

}

func (s *server) GetGCWhitelist(ctx context.Context, _ *empty.Empty) (*pb.Sessions, error) {
	sessions := []*pb.Session{}

	for id := range s.scheduler.GcWhiteList {
		sessions = append(sessions, &pb.Session{Id: id})
	}

	return &pb.Sessions{
		Sessions: sessions,
	}, nil
}

func (s *server) VerifyLiveLessonCompletion(ctx context.Context, uuid *pb.LessonUUID) (*pb.VerificationTask, error) {

	// Set verify flag in state and reject subsequent requests if they come in while this is locked.
	if _, ok := s.liveLessonState[uuid.Id]; !ok {
		return nil, errors.New("Livelesson not found")
	}
	ll := s.liveLessonState[uuid.Id]

	vtUUID := fmt.Sprintf("%s-%s", uuid.Id, ll.LessonStage)

	// If it already exists we can return it right away
	if vt, ok := s.verificationTasks[vtUUID]; ok {

		// If it's still working, just return it as-is
		if vt.Working {
			return vt, nil
		}

		// If it's not still working, regardless of success or failure, we can delete it from our queue
		// and return it's final state to the client.
		s.DeleteVerificationTask(vtUUID)
		return vt, nil

	}

	// Proceed with the creation of a new
	newVt := &pb.VerificationTask{
		LiveLesson: ll,
		Working:    true,
		Success:    false,
		Message:    "Starting verification",
	}
	s.SetVerificationTask(vtUUID, newVt)

	// Send via channel to scheduler

	return newVt, nil
}
