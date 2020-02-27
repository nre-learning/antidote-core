package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	copier "github.com/jinzhu/copier"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	db "github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

// RequestLiveSession generates a new session ID, performs basic security functions, installs the new session
// in state management, and returns the ID to the client
func (s *AntidoteAPI) RequestLiveSession(ctx context.Context, _ *empty.Empty) (*pb.LiveSession, error) {

	var sessionID string
	i := 0
	for {
		if i > 4 {
			return nil, errors.New("Unable to generate session ID")
		}
		sessionID = db.RandomID(10)
		_, err := s.Db.GetLiveSession(sessionID)
		if err == nil {
			i++
			continue
		}
		break
	}

	// TODO(mierdin): Need limit on sessions per IP, and livelessons per session. Both need to be configurable
	// and both livelessons.go and livesessions.go need appropriate checks before allocation

	md, _ := metadata.FromIncomingContext(ctx)
	// x-forwarded-host gets you IP+port, FWIW.
	forwardedFor := md["x-forwarded-for"]
	if len(forwardedFor) == 0 {
		return nil, errors.New("Unable to determine incoming IP address")
	}
	sourceIP := forwardedFor[0]
	log.Infof("Allocating session ID %s for incoming IP %s", sessionID, sourceIP)
	err := s.Db.CreateLiveSession(&models.LiveSession{
		ID:         sessionID,
		SourceIP:   sourceIP,
		Persistent: false,
	})
	if err != nil {
		return nil, errors.New("Unable to store new session record")
	}

	return &pb.LiveSession{ID: sessionID}, nil
}

// ListLiveSessions lists the currently available livesessions within the backing data store
func (s *AntidoteAPI) ListLiveSessions(ctx context.Context, _ *empty.Empty) (*pb.LiveSessions, error) {
	lsDBs, err := s.Db.ListLiveSessions()
	if err != nil {
		return nil, errors.New("Unable to list livesessions")
	}

	lsAPIs := map[string]*pb.LiveSession{}

	for _, lsdb := range lsDBs {
		lsAPIs[lsdb.ID] = liveSessionDBToAPI(&lsdb)
	}

	return &pb.LiveSessions{Items: lsAPIs}, nil
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
	copier.Copy(&pbLiveSession, liveSessionDB)
	return liveSessionDB
}
