package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	copier "github.com/jinzhu/copier"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	db "github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
	log "github.com/sirupsen/logrus"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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
	span := opentracing.StartSpan("api_livelesson_request", ext.SpanKindRPCClient)
	defer span.Finish()

	// Important to set these tags, since this span is likely to have many, many child spans
	// Get all the useful context embedded here at the top.
	span.SetTag("antidote_lesson_slug", lp.LessonSlug)
	span.SetTag("antidote_stage", lp.LessonStage)
	span.SetTag("antidote_session_id", lp.SessionId)
	md, _ := metadata.FromIncomingContext(ctx)
	// x-forwarded-host gets you IP+port, FWIW.
	forwardedFor := md["x-forwarded-for"]
	if len(forwardedFor) == 0 {
		span.LogEvent("Unable to determine source IP address")
	}
	span.SetTag("antidote_request_source_ip", forwardedFor[0])

	span.LogEvent("Received LiveLesson Request")

	if lp.SessionId == "" {
		msg := "Session ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	_, err := s.Db.GetLiveSession(span.Context(), lp.SessionId)
	if err != nil {
		// WARNING - the front-end relies on this string (minus the actual ID) to identify
		// if a new session ID should be requested. If you want to change this message, make sure
		// to change the front-end as well! Otherwise, the front-end won't know to request a valid
		// session ID before retrying the livelesson request
		msg := fmt.Sprintf("Invalid session ID %s", lp.SessionId)
		log.Error(msg)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	if lp.LessonSlug == "" {
		msg := "Lesson Slug cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	// TODO(mierdin): Ensure this works before removing entirely
	// if lp.LessonStage == 0 {
	// 	msg := "Stage ID cannot be nil"
	// 	log.Error(msg)
	// 	return nil, errors.New(msg)
	// }

	lesson, err := s.Db.GetLesson(span.Context(), lp.LessonSlug)
	if err != nil {
		log.Errorf("Couldn't find lesson slug '%s'", lp.LessonSlug)
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
	liveLessons, err := s.Db.ListLiveLessons(span.Context())
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

		// TODO(mierdin): Is this the right place for this? And is this being un-set somewhere?
		// Probably only return this if the user is trying to CHANGE to a new stage or something.
		// If they're not, just send them the existing livelesson state as if they refreshed the page.
		// if existingLL.Busy {   <---- getting rid of this, status is a suitable busy indicator
		if existingLL.Status != models.Status_READY {
			return &pb.LiveLessonId{}, errors.New("LiveLesson is currently busy. Can't make changes yet. Try later")
		}

		// If the incoming requested LessonStage is different from the current livelesson state,
		// tell the scheduler to change the state
		if existingLL.CurrentStage != lp.LessonStage {

			// Update state
			// TODO(Mierdin): Handle errors here
			s.Db.UpdateLiveLessonStatus(span.Context(), existingLL.ID, models.Status_CONFIGURATION)
			s.Db.UpdateLiveLessonStage(span.Context(), existingLL.ID, lp.LessonStage)
			s.Db.UpdateLiveLessonGuide(span.Context(), existingLL.ID, string(lesson.Stages[lp.LessonStage].GuideType), lesson.Stages[lp.LessonStage].GuideContents)

			span.LogEvent("Sending LiveLesson MODIFY request to scheduler")
			// Request the schedule move forward with stage change activities
			req := services.LessonScheduleRequest{
				Operation:     services.OperationType_MODIFY,
				Stage:         lp.LessonStage,
				LessonSlug:    lp.LessonSlug,
				LiveSessionID: lp.SessionId,
				LiveLessonID:  existingLL.ID,
				// APISpan:       span,
			}

			// Inject span context and send LSR into NATS
			var t services.TraceMsg
			if err := tracer.Inject(span.Context(), opentracing.Binary, &t); err != nil {
				log.Fatalf("%v for Inject.", err)
			}
			reqBytes, _ := json.Marshal(req)
			t.Write(reqBytes)
			s.NC.Publish("antidote.lsr.incoming", t.Bytes())

		} else {

			// Nothing to do but the user did interact with this lesson so we should boop it.
			req := services.LessonScheduleRequest{
				Operation:     services.OperationType_BOOP,
				LiveLessonID:  existingLL.ID,
				LessonSlug:    lp.LessonSlug,
				LiveSessionID: lp.SessionId,
			}

			// Inject span context and send LSR into NATS
			var t services.TraceMsg
			if err := tracer.Inject(span.Context(), opentracing.Binary, &t); err != nil {
				log.Fatalf("%v for Inject.", err)
			}
			reqBytes, _ := json.Marshal(req)
			t.Write(reqBytes)
			s.NC.Publish("antidote.lsr.incoming", t.Bytes())
		}

		return &pb.LiveLessonId{Id: existingLL.ID}, nil
	}

	// Initialize new LiveLesson
	newID := db.RandomID(10)

	newLL := &models.LiveLesson{
		ID:            newID,
		SessionID:     lp.SessionId,
		AntidoteID:    s.Config.InstanceID,
		LessonSlug:    lp.LessonSlug,
		GuideType:     string(lesson.Stages[lp.LessonStage].GuideType),
		LiveEndpoints: initializeLiveEndpoints(lesson),
		CurrentStage:  lp.LessonStage,
		Status:        models.Status_INITIALIZED,
		CreatedTime:   time.Now(),
	}

	if newLL.GuideType == "markdown" {
		newLL.GuideContents = lesson.Stages[lp.LessonStage].GuideContents
	}

	s.Db.CreateLiveLesson(span.Context(), newLL)

	req := services.LessonScheduleRequest{
		Operation:     services.OperationType_CREATE,
		Stage:         lp.LessonStage,
		LessonSlug:    lp.LessonSlug,
		LiveLessonID:  newID,
		LiveSessionID: lp.SessionId,
		Created:       time.Now(), // TODO(mierdin): Currently a lot of stuff uses this but you should probably remove it and point everything to the livelesson field
	}

	// Inject span context and send LSR into NATS
	var t services.TraceMsg
	if err := tracer.Inject(span.Context(), opentracing.Binary, &t); err != nil {
		log.Fatalf("%v for Inject.", err)
	}
	reqBytes, _ := json.Marshal(req)
	t.Write(reqBytes)
	s.NC.Publish("antidote.lsr.incoming", t.Bytes())

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
	span := opentracing.StartSpan("api_livelesson_create", ext.SpanKindRPCClient)
	defer span.Finish()

	err := s.Db.CreateLiveLesson(span.Context(), liveLessonAPIToDB(ll))
	if err != nil {
		return nil, err
	}
	// TODO(mierdin) the protobuf version doesn't have a timestamp so be sure to set this here on the db side.
	return &empty.Empty{}, nil
}

// GetLiveLesson retrieves a single LiveLesson via ID
func (s *AntidoteAPI) GetLiveLesson(ctx context.Context, llID *pb.LiveLessonId) (*pb.LiveLesson, error) {
	span := opentracing.StartSpan("api_livelesson_get", ext.SpanKindRPCClient)
	defer span.Finish()

	if llID.Id == "" {
		msg := "LiveLesson ID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	llDb, err := s.Db.GetLiveLesson(span.Context(), llID.Id)
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
	span := opentracing.StartSpan("api_livelesson_list", ext.SpanKindRPCClient)
	defer span.Finish()

	lls, err := s.Db.ListLiveLessons(span.Context())
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
	span := opentracing.StartSpan("api_livelesson_kill", ext.SpanKindRPCClient)
	defer span.Finish()

	ll, err := s.Db.GetLiveLesson(span.Context(), llID.Id)
	if err != nil {
		return nil, errors.New("livelesson not found")
	}

	// Send deletion request to scheduler. It will take care of sending the appropriate delete commands to
	// kubernetes, and if successful, removing the livelesson state.
	var t services.TraceMsg
	tracer := opentracing.GlobalTracer()
	if err := tracer.Inject(span.Context(), opentracing.Binary, &t); err != nil {
		log.Fatalf("%v for Inject.", err)
	}
	reqBytes, _ := json.Marshal(services.LessonScheduleRequest{
		Operation:    services.OperationType_DELETE,
		LiveLessonID: ll.ID,
	})
	t.Write(reqBytes)
	s.NC.Publish("antidote.lsr.incoming", t.Bytes())

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
	copier.Copy(&liveLessonDB, pbLiveLesson)

	liveLessonDB.LiveEndpoints = map[string]*models.LiveEndpoint{}

	for _, lep := range pbLiveLesson.LiveEndpoints {

		lepDB := &models.LiveEndpoint{}

		copier.Copy(&lepDB, lep)

		for _, lp := range lep.LivePresentations {
			lpDB := &models.LivePresentation{}
			copier.Copy(&lpDB, lp)
			lepDB.Presentations = append(lepDB.Presentations, lpDB)
		}

		liveLessonDB.LiveEndpoints[lep.Name] = lepDB

	}

	return liveLessonDB
}
