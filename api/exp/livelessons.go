package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	log "github.com/sirupsen/logrus"
)

func (s *SyringeAPIServer) RequestLiveLesson(ctx context.Context, lp *pb.LessonParams) (*pb.LessonUUID, error) {

	// Initialize span and populate with tags
	// tracer := opentracing.GlobalTracer()
	span := opentracing.GlobalTracer().StartSpan("livelesson_api_request", ext.SpanKindRPCClient)
	defer span.Finish()
	span.SetTag("antidote_lesson_id", lp.LessonId)
	span.SetTag("antidote_stage", lp.LessonStage)
	span.SetTag("antidote_session_id", lp.SessionId)

	span.LogEvent("Received LiveLesson Request")

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

	// A livelesson's UUID is formed with the lesson ID and the session ID together.
	// This allows us to store all livelessons within a flat key-value structure while maintaining
	// uniqueness.
	lessonUuid := fmt.Sprintf("%d-%s", lp.LessonId, lp.SessionId)

	// Identify lesson definition - return error if doesn't exist by ID
	if _, ok := s.Scheduler.Curriculum.Lessons[lp.LessonId]; !ok {
		log.Errorf("Couldn't find lesson ID %d", lp.LessonId)
		return &pb.LessonUUID{}, errors.New("Failed to find referenced lesson ID")
	}

	// Ensure requested stage is present. We add a zero-index stage on import to each lesson so that
	// stage ID 1 refers to the second index (1) in the stage slice.
	// So, to check that the requested stage exists, the length of the slice must be equal or greater than the
	// requested stage + 1. I.e. if there's only one stage, the slice will have a length of 2
	if len(s.Scheduler.Curriculum.Lessons[lp.LessonId].Stages) < int(lp.LessonStage) {
		msg := "Invalid stage ID for this lesson"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	// Check to see if the livelesson already exists in an errored state.
	// If so, clear it out so we can treat it like a new creation in the following logic.
	// TODO(mierdin): What if the namespace and resources still exist? Should we delete them first?
	// Maybe we should just return an error to the user and say "try again later"? Then let this get GC'd as normal?
	if s.LiveLessonExists(lessonUuid) {
		if s.LiveLessonState[lessonUuid].Error {
			s.DeleteLiveLesson(lessonUuid)
		}
	}

	// Check to see if it already exists in memory. If it does, don't send provision request.
	// Just look it up and send UUID
	if s.LiveLessonExists(lessonUuid) {

		if s.LiveLessonState[lessonUuid].LessonStage != lp.LessonStage {

			// Update in-memory state
			s.UpdateLiveLessonStage(lessonUuid, lp.LessonStage)

			// Request the schedule move forward with stage change activities
			span.LogEvent("Sending LiveLesson MODIFY request to scheduler")
			req := &scheduler.LessonScheduleRequest{
				Lesson:    s.Scheduler.Curriculum.Lessons[lp.LessonId],
				Operation: scheduler.OperationType_MODIFY,
				Stage:     lp.LessonStage,
				Uuid:      lessonUuid,
				Session:   lp.SessionId,
				APISpan:   span,
			}

			s.Scheduler.Requests <- req

		} else {

			// Nothing to do but the user did interact with this lesson so we should boop it.
			span.LogEvent("Sending LiveLesson BOOP request to scheduler")
			req := &scheduler.LessonScheduleRequest{
				Operation: scheduler.OperationType_BOOP,
				Uuid:      lessonUuid,
				Session:   lp.SessionId,
				Lesson:    s.Scheduler.Curriculum.Lessons[lp.LessonId],
				APISpan:   span,
			}

			s.Scheduler.Requests <- req
		}

		return &pb.LessonUUID{Id: lessonUuid}, nil
	}

	// 3 - if doesn't already exist, put together schedule request and send to channel
	span.LogEvent("Sending LiveLesson CREATE request to scheduler")
	req := &scheduler.LessonScheduleRequest{
		Lesson:    s.Scheduler.Curriculum.Lessons[lp.LessonId],
		Operation: scheduler.OperationType_CREATE,
		Stage:     lp.LessonStage,
		Session:   lp.SessionId,
		Uuid:      lessonUuid,
		APISpan:   span,
		Created:   time.Now(),
	}
	s.Scheduler.Requests <- req

	// Pre-emptively populate livelessons map with initial status.
	// This will be updated when the Scheduler response comes back.
	s.SetLiveLesson(lessonUuid, &pb.LiveLesson{
		LiveLessonStatus: pb.Status_INITIAL_BOOT,
		LessonId:         lp.LessonId,
		LessonUUID:       lessonUuid,
		LessonStage:      lp.LessonStage,
	})

	return &pb.LessonUUID{Id: lessonUuid}, nil
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
