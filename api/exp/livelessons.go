package api

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	copier "github.com/jinzhu/copier"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	db "github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
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
func (s *AntidoteAPI) RequestLiveLesson(ctx context.Context, lp *pb.LiveLessonRequest) (*pb.LiveLessonId, error) {

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
			req := services.LessonScheduleRequest{
				Operation:     services.OperationType_MODIFY,
				Stage:         lp.LessonStage,
				LessonSlug:    lp.LessonSlug,
				LiveSessionID: lp.SessionId,
				LiveLessonID:  existingLL.ID,
			}
			s.Requests <- req

		} else {

			// Nothing to do but the user did interact with this lesson so we should boop it.
			req := services.LessonScheduleRequest{
				Operation:     services.OperationType_BOOP,
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
		CreatedTime:   time.Now(),
	})

	req := services.LessonScheduleRequest{
		Operation:     services.OperationType_CREATE,
		Stage:         lp.LessonStage,
		LessonSlug:    lp.LessonSlug,
		LiveLessonID:  newID,
		LiveSessionID: lp.SessionId,
		Created:       time.Now(), // TODO(mierdin): Currently a lot of stuff uses this but you should probably remove it and point everything to the livelesson field
	}
	s.Requests <- req

	return &pb.LiveLessonId{Id: newID}, nil
}

func initializeLiveEndpoints(lesson *models.Lesson) map[string]*models.LiveEndpoint {
	liveEps := map[string]*models.LiveEndpoint{}

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

		liveEps[lep.Name] = lep

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
func (s *AntidoteAPI) HealthCheck(ctx context.Context, _ *empty.Empty) (*pb.LBHealthCheckResponse, error) {
	return &pb.LBHealthCheckResponse{}, nil
}

// CreateLiveLesson is a HIGHLY non-production function for inserting livelesson state directly for debugging
// or test purposes. Use this at your own peril.
func (s *AntidoteAPI) CreateLiveLesson(ctx context.Context, ll *pb.LiveLesson) (*empty.Empty, error) {

	llDB := liveLessonAPIToDB(ll)

	s.Db.CreateLiveLesson(llDB)

	// the protobuf version doesn't have a timestamp so be sure to set this here on the db side.
	return &empty.Empty{}, nil
}

// GetLiveLesson retrieves a single LiveLesson via ID
func (s *AntidoteAPI) GetLiveLesson(ctx context.Context, llID *pb.LiveLessonId) (*pb.LiveLesson, error) {

	if llID.Id == "" {
		msg := "LiveLesson ID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	llDb, err := s.Db.GetLiveLesson(llID.Id)
	if err != nil {
		return nil, errors.New("livelesson not found")
	}

	// TODO(mierdin): Is this the right thing to do?
	if llDb.Error {
		return nil, errors.New("Livelesson encountered errors during provisioning. See antidote logs")
	}

	return liveLessonDBToAPI(llDb), nil
}

// ListLiveLessons returns a list of LiveLessons present in the data store
func (s *AntidoteAPI) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessons, error) {

	lls, err := s.Db.ListLiveLessons()
	if err != nil {
		return nil, errors.New("Unable to list livelessons")
	}

	pblls := map[string]*pb.LiveLesson{}

	for _, ll := range lls {
		pblls[ll.ID] = liveLessonDBToAPI(&ll)
	}

	return &pb.LiveLessons{LiveLessons: pblls}, nil
}

// KillLiveLesson allows a client to request that a livelesson is killed, meaning the underlying
// resources (like kubernetes namespace) are deleted, and local state is cleaned up appropriately
func (s *AntidoteAPI) KillLiveLesson(ctx context.Context, llID *pb.LiveLessonId) (*pb.KillLiveLessonStatus, error) {

	ll, err := s.Db.GetLiveLesson(llID.Id)
	if err != nil {
		return nil, errors.New("livelesson not found")
	}

	// Send deletion request to scheduler. It will take care of sending the appropriate delete commands to
	// kubernetes, and if successful, removing the livelesson state.
	s.Requests <- services.LessonScheduleRequest{
		Operation:    services.OperationType_DELETE,
		LiveLessonID: ll.ID,
	}

	// TODO(mierdin): May want to clarify that this is a successful receipt of the kill command - not
	//that it was actually killed.
	return &pb.KillLiveLessonStatus{Success: true}, nil
}

// liveLessonDBToAPI translates a single LiveLesson from the `db` package models into the
// api package's equivalent
func liveLessonDBToAPI(dbLL *models.LiveLesson) *pb.LiveLesson {

	// Copy the majority of like-named fields
	var llAPI pb.LiveLesson
	copier.Copy(&llAPI, dbLL)

	// Handle the one-offs
	llAPI.Status = string(dbLL.Status)

	eps := map[string]*pb.LiveEndpoint{}

	for k, v := range dbLL.LiveEndpoints {
		var lep pb.LiveEndpoint
		copier.Copy(&lep, v)

		for _, c := range v.Presentations {

			var lp pb.LivePresentation

			lp.Name = c.Name
			lp.Type = string(c.Type)
			lp.Port = c.Port

			lep.LivePresentations = append(lep.LivePresentations, &lp)
		}

		eps[k] = &lep
	}

	llAPI.LiveEndpoints = eps

	return &llAPI
}

// liveLessonAPIToDB translates a single LiveLesson from the `api` package models into the
// `db` package's equivalent
func liveLessonAPIToDB(pbLiveLesson *pb.LiveLesson) *models.LiveLesson {
	liveLessonDB := &models.LiveLesson{}
	copier.Copy(&pbLiveLesson, liveLessonDB)
	return liveLessonDB
}
