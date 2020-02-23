package api

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	db "github.com/nre-learning/syringe/db"
	models "github.com/nre-learning/syringe/db/models"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

func (s *SyringeAPIServer) RequestLiveLesson(ctx context.Context, lp *pb.LiveLessonRequest) (*pb.LessonUUID, error) {

	// TODO(mierdin) look up session ID in DB first
	if lp.SessionId == "" {
		msg := "Session ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	if lp.LessonSlug == "" {
		msg := "Lesson Slug cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	if lp.LessonStage == 0 {
		msg := "Stage ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	lesson, err := s.Db.GetLesson(lp.LessonSlug)
	if err != nil {
		log.Errorf("Couldn't find lesson slug %d", lp.LessonSlug)
		return nil, errors.New("Failed to find referenced lesson slug")
	}

	// Ensure requested stage is present
	if len(lesson.Stages) < 1+int(lp.LessonStage) {
		msg := "Invalid stage ID for this lesson"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	// Determine if a livelesson exists that matches this session and lesson
	llExists := false
	var existingLL *models.LiveLesson
	liveLessons, err := s.Db.ListLiveLessons()
	for _, ll := range liveLessons {
		if ll.SessionID == lp.SessionId && ll.LessonSlug == lp.LessonSlug {
			existingLL = &ll
			llExists = true
		}
	}

	// LiveLesson already exists, so we should handle this accordingly
	if llExists {

		// Check to see if the livelesson already exists in an errored state.
		// If so, return error. TODO(mierdin): Consider this further, maybe we can do something to clean up the
		// LiveLesson and the underlying resources and try again?
		if existingLL.Error {
			return &pb.LiveLessonResponse{}, errors.New("Lesson is in errored state. Please try again later")
		}

		// TODO(mierdin): Is this the right place for this?
		if existingLL.Busy {
			return &pb.LiveLessonResponse{}, errors.New("LiveLesson is currently busy")
		}

		// If the incoming requested LessonStage is different from the current livelesson state,
		// tell the scheduler to change the state
		if existingLL.LessonStage != lp.LessonStage {

			// Update state
			existingLL.LessonStage = lp.LessonStage
			existingLL.Busy = true
			s.Db.UpdateLiveLesson(existingLL)

			// Request the schedule move forward with stage change activities
			req := &scheduler.LessonScheduleRequest{
				Operation:     scheduler.OperationType_MODIFY,
				Stage:         lp.LessonStage,
				LessonSlug:    lp.LessonSlug,
				LiveSessionID: lp.SessionId,
				LiveLessonID:  existingLL.ID,
			}
			s.Requests <- req

		} else {

			// Nothing to do but the user did interact with this lesson so we should boop it.
			req := &scheduler.LessonScheduleRequest{
				Operation:     scheduler.OperationType_BOOP,
				LiveLessonID:  existingLL.ID,
				LessonSlug:    lp.LessonSlug,
				LiveSessionID: lp.SessionId,
			}
			s.Requests <- req
		}

		return &pb.LiveLessonResponse{Id: existingLL.ID}, nil
	}

	// Initialize new LiveLesson
	// TODO(mierdin): there's much more to do here. Fill out endpoints with ports, etc
	newID := db.RandomID(10)
	s.Db.CreateLiveLesson(&models.LiveLesson{
		ID:            newID,
		SessionID:     lp.SessionId,
		LessonSlug:    lp.LessonSlug,
		LiveEndpoints: map[string]*models.LiveEndpoint{},
		LessonStage:   lp.LessonStage,
		Busy:          true,
		Status:        models.Status_INITIALIZED,
		// CreatedTime:   time.Now(),
	})

	req := &scheduler.LessonScheduleRequest{
		Operation:     scheduler.OperationType_CREATE,
		Stage:         lp.LessonStage,
		LessonSlug:    lp.LessonSlug,
		LiveLessonID:  newID,
		LiveSessionID: lp.SessionId,
		Created:       time.Now(),
	}
	s.Requests <- req

	return &pb.LiveLessonResponse{Id: newID}, nil
}

// HealthCheck provides an endpoint for retuning 200K for load balancer health checks
func (s *SyringeAPIServer) HealthCheck(ctx context.Context, _ *empty.Empty) (*pb.HealthCheckMessage, error) {
	return &pb.LBHealthCheckResponse{}, nil
}

// GetLiveLesson retrieves a single LiveLesson via ID
func (s *SyringeAPIServer) GetLiveLesson(ctx context.Context, llID *pb.LiveLessonId) (*pb.LiveLesson, error) {

	if llID.Id == "" {
		msg := "LiveLesson ID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	ll, err := s.Db.GetLiveLesson(llID.Id)
	if err != nil {
		return nil, errors.New("livelesson not found")
	}

	if ll.Error {
		return nil, errors.New("Livelesson encountered errors during provisioning. See syringe logs")
	}
	return ll, nil

}

func (s *SyringeAPIServer) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessons, error) {
	return &pb.LiveLessons{}, nil
}

func (s *SyringeAPIServer) KillLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.KillLiveLessonStatus, error) {

	// TODO(mierdin): Need to implement

	// if _, ok := s.LiveLessonState[uuid.Id]; !ok {
	// 	return nil, errors.New("Livelesson not found")
	// }

	// s.Scheduler.Requests <- &scheduler.LessonScheduleRequest{
	// 	Operation: scheduler.OperationType_DELETE,
	// 	Uuid:      uuid.Id,
	// }

	return &pb.KillLiveLessonStatus{Success: true}, nil
}
