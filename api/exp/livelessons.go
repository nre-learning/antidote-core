package api

import (
	"context"
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
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

	// Identify lesson definition - return error if doesn't exist by ID
	if _, ok := s.scheduler.LessonDefs[lp.LessonId]; !ok {
		log.Errorf("Couldn't find lesson ID %d", lp.LessonId)
		return &pb.LessonUUID{}, errors.New("Failed to find referenced lesson ID")
	}

	// Get lessonStage from incoming request (or set to 1 by default if not specified)
	var lessonStage int32 = 1
	if lp.LessonStage != 0 {
		lessonStage = lp.LessonStage
	}

	// Check to see if it already exists in memory. If it does, don't send provision request.
	// Just look it up and send UUID
	log.Infof("Looking up session %s", lp.SessionId)
	if _, ok := s.sessions[lp.SessionId]; ok {
		if lessonUuid, ok := s.sessions[lp.SessionId][lp.LessonId]; ok {

			log.Debugf("Found existing session %s", lp.SessionId)

			log.Debugf("Current lessonStage: %d - new lessonStage: %d", s.liveLessons[lessonUuid].LessonStage, lessonStage)

			if s.liveLessons[lessonUuid].LessonStage != lessonStage {

				// Since this already existed, we don't need to update the sessions map, or the livelessons map
				// Just update the stage and ready properties before sending modify request
				s.liveLessons[lessonUuid].LessonStage = lessonStage
				s.liveLessons[lessonUuid].Ready = false

				s.scheduler.Requests <- &scheduler.LessonScheduleRequest{
					LessonDef: s.scheduler.LessonDefs[lp.LessonId],
					Operation: scheduler.OperationType_MODIFY,
					Stage:     lessonStage,
					Uuid:      lessonUuid,
					Session:   lp.SessionId,
				}

			}

			return &pb.LessonUUID{Id: lessonUuid}, nil
		} else {
			log.Infof("session ID found but not for this lesson: %d", lp.LessonId)
		}
	} else {

		// Doesn't exist, prep this spot with an empty map
		s.sessions[lp.SessionId] = map[int32]string{}
	}

	// Generate UUID, make sure it doesn't conflict with another (unlikely but easy to check)
	var newUuid string
	for {
		newUuid = GenerateUUID()
		if _, ok := s.liveLessons[newUuid]; !ok {
			break
		}
	}

	// TODO(mierdin): consider not having any tables in memory at all. Just make everything function off of namespace names
	// and literally store all state in kubernetes
	//
	// Ensure sessions table is updated with the new session
	s.sessions[lp.SessionId][lp.LessonId] = newUuid

	// 3 - if doesn't already exist, put together schedule request and send to channel
	s.scheduler.Requests <- &scheduler.LessonScheduleRequest{
		LessonDef: s.scheduler.LessonDefs[lp.LessonId],
		Operation: scheduler.OperationType_CREATE,
		Stage:     lessonStage,
		Uuid:      newUuid,
		Session:   lp.SessionId,
	}

	// Pre-emptively populate livelessons map with non-ready livelesson.
	// This will be updated when the scheduler response comes back.
	s.liveLessons[newUuid] = &pb.LiveLesson{Ready: false, LessonId: lp.LessonId, LessonUUID: newUuid, LessonStage: lessonStage}
	log.Infof("LiveLessons map: %v", s.liveLessons)

	return &pb.LessonUUID{Id: newUuid}, nil
}

func (s *server) SetLiveLesson(ctx context.Context, lp *pb.LessonParams) (*pb.LessonUUID, error) {

	// Need to set Ready to false immediately before returning to avoid race conditions
	uuid := s.sessions[lp.SessionId][lp.LessonId]
	s.liveLessons[uuid].Ready = false

	// TODO(mierdin): Finish the implementation, sending a schedule request.
	// NEED TO make sure the referenced lesson doesn't use the shared topology

	return &pb.LessonUUID{Id: uuid}, nil
}

func (s *server) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessons, error) {
	return &pb.LiveLessons{}, nil
}

func (s *server) GetLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.LiveLesson, error) {

	if uuid.Id == "" {
		msg := "Lesson UUID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	log.Infof("Looking up livelesson %s", uuid.Id)
	if _, ok := s.liveLessons[uuid.Id]; !ok {
		return nil, errors.New("livelesson not found")
	}

	log.Debug("CURRENT LIVELESSONS")
	log.Debug(s.liveLessons)

	log.Debugf("About to return %s", s.liveLessons[uuid.Id])
	return s.liveLessons[uuid.Id], nil

}
