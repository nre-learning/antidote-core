package api

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	copier "github.com/jinzhu/copier"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	db "github.com/nre-learning/syringe/db"
	models "github.com/nre-learning/syringe/db/models"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

// RequestLiveLesson allows the client to get a handle on a livelesson when they only know their session ID
// and the desired lesson slug and stage ID. This is typically the first call on antidote-web's lab interface, since these
// pieces of information are typically known, but a specific livelesson ID is not. There are three main reasons
// why you would want to call this endpoint:
//
// - Request that a livelesson is created, such as when you initially load a lesson
// - Modify an existing livelesson - such as moving to a different stage ID
// - Refreshing the lastUsed timestamp on the livelesson, which lets the back-end know it's still in use.
func (s *SyringeAPIServer) RequestLiveLesson(ctx context.Context, lp *pb.LiveLessonRequest) (*pb.LiveLessonId, error) {

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
			return &pb.LiveLessonId{}, errors.New("Lesson is in errored state. Please try again later")
		}

		// TODO(mierdin): Is this the right place for this?
		if existingLL.Busy {
			return &pb.LiveLessonId{}, errors.New("LiveLesson is currently busy")
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

		return &pb.LiveLessonId{Id: existingLL.ID}, nil
	}

	// Initialize new LiveLesson
	newID := db.RandomID(10)
	s.Db.CreateLiveLesson(&models.LiveLesson{
		ID:            newID,
		SessionID:     lp.SessionId,
		LessonSlug:    lp.LessonSlug,
		LiveEndpoints: initializeLiveEndpoints(lesson),
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

	return &pb.LiveLessonId{Id: newID}, nil
}

func initializeLiveEndpoints(lesson *models.Lesson) []*models.LiveEndpoint {
	liveEps := []*models.LiveEndpoint{}

	for i := range lesson.Endpoints {
		ep := lesson.Endpoints[i]

		lep := &models.LiveEndpoint{
			Name:              ep.Name,
			Image:             ep.Image,
			Ports:             []int32{},
			Presentations:     []*models.LivePresentation{},
			ConfigurationType: ep.ConfigurationType,
			Host:              "", // Will be populated by scheduler once service is created for this endpoint
		}

		for p := range ep.Presentations {
			lep.Presentations = append(lep.Presentations, &models.LivePresentation{
				Name: ep.Presentations[p].Name,
				Port: ep.Presentations[p].Port,
				Type: ep.Presentations[p].Type,
			})

			lep.Ports = append(lep.Ports, ep.Presentations[p].Port)
		}

		for pt := range ep.AdditionalPorts {
			lep.Ports = append(lep.Ports, ep.AdditionalPorts[pt])
		}

		lep.Ports = unique(lep.Ports)

		liveEps = append(liveEps, lep)

		// the flattened "Ports" property will be used by the scheduler to know what ports to open
		// in the kubernetes service. Once it does this, the scheduler will update the "Host" property with
		// the IP address to use for the endpoint (all ports)

	}
	return liveEps
}

func unique(intSlice []int32) []int32 {
	keys := make(map[int32]bool)
	list := []int32{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// HealthCheck provides an endpoint for retuning 200K for load balancer health checks
func (s *SyringeAPIServer) HealthCheck(ctx context.Context, _ *empty.Empty) (*pb.LBHealthCheckResponse, error) {
	return &pb.LBHealthCheckResponse{}, nil
}

// GetLiveLesson retrieves a single LiveLesson via ID
func (s *SyringeAPIServer) GetLiveLesson(ctx context.Context, llID *pb.LiveLessonId) (*pb.LiveLesson, error) {

	if llID.Id == "" {
		msg := "LiveLesson ID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	llDb, err := s.Db.GetLiveLesson(llID.Id)
	if err != nil {
		return nil, errors.New("livelesson not found")
	}

	if llDb.Error {
		return nil, errors.New("Livelesson encountered errors during provisioning. See syringe logs")
	}

	return liveLessonDBToAPI(llDb), nil
}

// ListLiveLessons returns a list of LiveLessons present in the data store
func (s *SyringeAPIServer) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessons, error) {
	return &pb.LiveLessons{}, nil
}

// KillLiveLesson allows a client to request that a livelesson is killed, meaning the underlying
// resources (like kubernetes namespace) are deleted, and local state is cleaned up appropriately
func (s *SyringeAPIServer) KillLiveLesson(ctx context.Context, llID *pb.LiveLessonId) (*pb.KillLiveLessonStatus, error) {

	ll, err := s.Db.GetLiveLesson(llID.Id)
	if err != nil {
		return nil, errors.New("livelesson not found")
	}

	// Send deletion request to scheduler. It will take care of sending the appropriate delete commands to
	// kubernetes, and if successful, removing the livelesson state.
	s.Requests <- &scheduler.LessonScheduleRequest{
		Operation:    scheduler.OperationType_DELETE,
		LiveLessonID: ll.ID,
	}

	// TODO(mierdin): May want to clarify that this is a successful receipt of the kill command - not
	//that it was actually killed.
	return &pb.KillLiveLessonStatus{Success: true}, nil
}

// liveLessonDBToAPI translates a single LiveLesson from the `db` package models into the
// api package's equivalent
func liveLessonDBToAPI(dbLiveLesson *models.LiveLesson) *pb.LiveLesson {
	liveLessonAPI := &pb.LiveLesson{}
	copier.Copy(&liveLessonAPI, dbLiveLesson)
	return liveLessonAPI
}

// liveLessonAPIToDB translates a single LiveLesson from the `api` package models into the
// `db` package's equivalent
func liveLessonAPIToDB(pbLiveLesson *pb.LiveLesson) *models.LiveLesson {
	liveLessonDB := &models.LiveLesson{}
	copier.Copy(&pbLiveLesson, liveLessonDB)
	return liveLessonDB
}
