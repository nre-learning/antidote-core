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

	// TODO(mierdin): need to handle invalid stage
	if _, ok := s.scheduler.LessonDefs[lp.LessonId].Stages[lp.LessonStage]; !ok {
		msg := "Lesson ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	var lessonStage int32 = 1
	// If the incoming stageID is valid, we can use it. Otherwise, leave it at 1
	if _, ok := s.scheduler.LessonDefs[lp.LessonId].Stages[lp.LessonStage]; ok {
		lessonStage = lp.LessonStage
	}

	// Check to see if it already exists in memory. If it does, don't send provision request.
	// Just look it up and send UUID
	if _, ok := s.sessions[lp.SessionId]; ok {
		if lessonUuid, ok := s.sessions[lp.SessionId][lp.LessonId]; ok {

			if s.liveLessons[lessonUuid].LessonStage != lessonStage {

				// 10.32.0.16 - - [28/Sep/2018:23:21:17 +0000] "POST /exp/livelesson HTTP/1.1" 408 66
				// panic: runtime error: invalid memory address or nil pointer dereference
				// [signal SIGSEGV: segmentation violation code=0x1 addr=0x30 pc=0xf4f88a]
				// goroutine 342 [running]:
				// github.com/nre-learning/syringe/api/exp.(*server).RequestLiveLesson(0xc4201d0020, 0x1370700, 0xc4204563f0, 0xc4204d3100, 0xc4201d0020, 0xc420456330, 0x1102de0)
				// 	/go/src/github.com/nre-learning/syringe/api/exp/livelessons.go:53 +0x86a
				// github.com/nre-learning/syringe/api/exp/generated._LiveLessonsService_RequestLiveLesson_Handler(0x119d680, 0xc4201d0020, 0x1370700, 0xc4204563f0, 0xc4201de230, 0x0, 0x0, 0x0, 0xc4200eecf8, 0xf48563)
				// 	/go/src/github.com/nre-learning/syringe/api/exp/generated/livelesson.pb.go:631 +0x241
				// github.com/nre-learning/syringe/vendor/google.golang.org/grpc.(*Server).processUnaryRPC(0xc4201ca000, 0x137c420, 0xc420260000, 0xc42051ed00, 0xc4203de180, 0x1c775b8, 0x0, 0x0, 0x0)
				// 	/go/src/github.com/nre-learning/syringe/vendor/google.golang.org/grpc/server.go:1011 +0x4fc
				// github.com/nre-learning/syringe/vendor/google.golang.org/grpc.(*Server).handleStream(0xc4201ca000, 0x137c420, 0xc420260000, 0xc42051ed00, 0x0)
				// 	/go/src/github.com/nre-learning/syringe/vendor/google.golang.org/grpc/server.go:1249 +0x1318
				// github.com/nre-learning/syringe/vendor/google.golang.org/grpc.(*Server).serveStreams.func1.1(0xc4203b60c0, 0xc4201ca000, 0x137c420, 0xc420260000, 0xc42051ed00)
				// 	/go/src/github.com/nre-learning/syringe/vendor/google.golang.org/grpc/server.go:680 +0x9f
				// created by github.com/nre-learning/syringe/vendor/google.golang.org/grpc.(*Server).serveStreams.func1
				// 	/go/src/github.com/nre-learning/syringe/vendor/google.golang.org/grpc/server.go:678 +0xa1

				// Since this already existed, we don't need to update the sessions map, or the livelessons map
				// Just update the stage and ready properties before sending modify request
				s.liveLessons[lessonUuid].LessonStage = lessonStage
				s.liveLessons[lessonUuid].Ready = false

				req := &scheduler.LessonScheduleRequest{
					LessonDef: s.scheduler.LessonDefs[lp.LessonId],
					Operation: scheduler.OperationType_MODIFY,
					Stage:     lessonStage,
					Uuid:      lessonUuid,
					Session:   lp.SessionId,
				}

				s.scheduler.Requests <- req

				// TODO(mierdin): Need to make this async, to not impact UX
				// s.recordRequestTSDB(req)

			} else {

				// Nothing to do but the user did interact with this lesson so we should boop it.
				req := &scheduler.LessonScheduleRequest{
					LessonDef: s.scheduler.LessonDefs[lp.LessonId],
					Operation: scheduler.OperationType_BOOP,
					Stage:     0,
					Uuid:      "",
					Session:   lp.SessionId,
				}

				s.scheduler.Requests <- req

				// TODO(mierdin): Need to make this async, to not impact UX
				// s.recordRequestTSDB(req)
			}

			return &pb.LessonUUID{Id: lessonUuid}, nil
		} else {
			log.Infof("session ID found but not for this lesson: %d", lp.LessonId)
		}
	} else {

		// Doesn't exist, prep this spot with an empty map
		log.Infof("Creating new session: %s", lp.SessionId)
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
	req := &scheduler.LessonScheduleRequest{
		LessonDef: s.scheduler.LessonDefs[lp.LessonId],
		Operation: scheduler.OperationType_CREATE,
		Stage:     lessonStage,
		Uuid:      newUuid,
		Session:   lp.SessionId,
	}
	s.scheduler.Requests <- req

	// TODO(mierdin): Need to make this async, to not impact UX
	// s.recordRequestTSDB(req)

	// Pre-emptively populate livelessons map with non-ready livelesson.
	// This will be updated when the scheduler response comes back.
	s.liveLessons[newUuid] = &pb.LiveLesson{Ready: false, LessonId: lp.LessonId, LessonUUID: newUuid, LessonStage: lessonStage}
	log.Infof("LiveLessons map: %v", s.liveLessons)

	return &pb.LessonUUID{Id: newUuid}, nil
}

func (s *server) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessonMap, error) {

	// if _, ok := s.sessions[lp.SessionId]; ok {
	// 	if lessonUuid, ok := s.sessions[lp.SessionId][lp.LessonId]; ok {

	llm := pb.LiveLessonMap{}
	// Sessions: make(map[int]pb.LessontoUUIDMap{
	// 	Uuids: make(map[int32]pb.UUIDtoLiveLessonMap{
	// 		LiveLessons: make(map[string]pb.LiveLesson{}),
	// 	}),
	// }),
	// }
	// liveLessons: make(map[string]*pb.LiveLesson),
	// sessions:    make(map[string]map[int32]string),
	llm.Sessions = make(map[string]*pb.LessontoUUIDMap)
	for sessionId, lessons := range s.sessions {
		llm.Sessions[sessionId] = &pb.LessontoUUIDMap{}
		llm.Sessions[sessionId].Uuids = make(map[int32]*pb.UUIDtoLiveLessonMap)
		for lessonId, uuid := range lessons {
			llm.Sessions[sessionId].Uuids[lessonId] = &pb.UUIDtoLiveLessonMap{
				Livelessons: map[string]*pb.LiveLesson{
					uuid: s.liveLessons[uuid],
				},
			}

		}
	}

	return &llm, nil
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

	if _, ok := s.liveLessons[uuid.Id]; !ok {
		return nil, errors.New("livelesson not found")
	}

	// Remove all blackbox entries
	ll := s.liveLessons[uuid.Id]
	newEndpoints := []*pb.Endpoint{}
	for e := range ll.Endpoints {
		if ll.Endpoints[e].Type != pb.Endpoint_BLACKBOX {
			newEndpoints = append(newEndpoints, ll.Endpoints[e])
		}
	}
	ll.Endpoints = newEndpoints

	return ll, nil

}
