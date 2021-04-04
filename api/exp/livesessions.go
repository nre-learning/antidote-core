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
	ot "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	"google.golang.org/grpc/metadata"
)

// RequestLiveSession generates a new session ID, performs basic security functions, installs the new session
// in state management, and returns the ID to the client
func (s *AntidoteAPI) RequestLiveSession(ctx context.Context, _ *empty.Empty) (*pb.LiveSession, error) {
	span := ot.StartSpan("api_livesession_request", ext.SpanKindRPCClient)
	defer span.Finish()

	md, _ := metadata.FromIncomingContext(ctx)
	// x-forwarded-host gets you IP+port, FWIW.
	forwardedFor := md["x-forwarded-for"]
	if len(forwardedFor) == 0 {
		return nil, errors.New("Unable to determine incoming IP address")
	}
	sourceIP := forwardedFor[0]
	span.SetTag("antidote_request_source_ip", sourceIP)

	// Figure out how many sessions this IP has opened, if enabled
	// (limit must be > 0)
	if s.Config.LiveSessionLimit > 0 {
		lsList, _ := s.Db.ListLiveSessions(span.Context())
		lsCount := 0
		for _, ls := range lsList {
			if ls.SourceIP == sourceIP {
				lsCount++
			}
		}
		if lsCount >= s.Config.LiveSessionLimit {
			span.LogFields(
				log.String("sourceIP", sourceIP),
				log.Int("lsCount", lsCount),
			)
			ext.Error.Set(span, true)
			return nil, errors.New("This IP address has exceeded the maximum number of liveSessions")
		}
	}

	var sessionID string
	i := 0
	for {
		if i > 4 {
			return nil, errors.New("Unable to generate session ID")
		}
		sessionID = db.RandomID(10)
		_, err := s.Db.GetLiveSession(span.Context(), sessionID)
		if err == nil {
			i++
			continue
		}
		break
	}

	if s.Config.DevMode {
		sessionID = "antidotedevmode"
	}

	span.LogFields(log.String("allocatedSessionId", sessionID))
	err := s.Db.CreateLiveSession(span.Context(), &models.LiveSession{
		ID:          sessionID,
		SourceIP:    sourceIP,
		Persistent:  false,
		CreatedTime: time.Now(),
	})
	if err != nil {
		return nil, errors.New("Unable to store new session record")
	}

	return &pb.LiveSession{ID: sessionID}, nil
}

// CreateLiveSession is a HIGHLY non-production function for inserting livesession state directly
// for debugging or test purposes. Use this at your own peril.
func (s *AntidoteAPI) CreateLiveSession(ctx context.Context, ls *pb.LiveSession) (*empty.Empty, error) {
	span := ot.StartSpan("api_livesession_create", ext.SpanKindRPCClient)
	defer span.Finish()

	lsDB := liveSessionAPIToDB(ls)
	lsDB.CreatedTime = time.Now()

	err := s.Db.CreateLiveSession(span.Context(), lsDB)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// GetLiveSession fetches the LiveSession assocciated with the sessionID
func (s *AntidoteAPI) GetLiveSession(ctx context.Context, ls *pb.LiveSession) (*pb.LiveSession, error) {
	span := ot.StartSpan("api_livesession_get", ext.SpanKindRPCClient)
	defer span.Finish()

	lsDB, err := s.Db.GetLiveSession(span.Context(), ls.ID)
	if err != nil {
		return nil, errors.New("livesession not found")
	}
	return liveSessionDBToAPI(&lsDB), nil
}

// ListLiveSessions lists the currently available livesessions within the backing data store
func (s *AntidoteAPI) ListLiveSessions(ctx context.Context, _ *empty.Empty) (*pb.LiveSessions, error) {
	span := ot.StartSpan("api_livesession_list", ext.SpanKindRPCClient)
	defer span.Finish()

	lsDBs, err := s.Db.ListLiveSessions(span.Context())
	if err != nil {
		return nil, errors.New("Unable to list livesessions")
	}

	lsAPIs := map[string]*pb.LiveSession{}

	for _, lsdb := range lsDBs {
		lsAPIs[lsdb.ID] = liveSessionDBToAPI(&lsdb)
	}

	return &pb.LiveSessions{Items: lsAPIs}, nil
}

// UpdateLiveSessionPersistence updates the persistence flag in the session database
func (s *AntidoteAPI) UpdateLiveSessionPersistence(ctx context.Context, persistence *pb.SessionPersistence) (*empty.Empty, error) {
	span := ot.StartSpan("api_livesession_persist", ext.SpanKindRPCClient)
	defer span.Finish()

	err := s.Db.UpdateLiveSessionPersistence(span.Context(), persistence.SessionID, persistence.Persistent)
	if err != nil {
		return &empty.Empty{}, err
	}

	return &empty.Empty{}, nil
}

// liveSessionDBToAPI translates a single LiveSession from the `db` package models into the
// api package's equivalent
func liveSessionDBToAPI(dbLS *models.LiveSession) *pb.LiveSession {

	// Copy the majority of like-named fields
	var lsAPI pb.LiveSession
	copier.Copy(&lsAPI, dbLS)

	return &lsAPI
}

// liveSessionAPIToDB translates a single LiveSession from the `api` package models into the
// `db` package's equivalent
func liveSessionAPIToDB(pbLiveSession *pb.LiveSession) *models.LiveSession {
	liveSessionDB := &models.LiveSession{}
	copier.Copy(&liveSessionDB, pbLiveSession)
	return liveSessionDB
}
