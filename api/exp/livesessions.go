package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	db "github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
)

// RequestLiveSession generates a new session ID, performs basic security functions, installs the new session
// in state management, and returns the ID to the client
func (s *AntidoteAPI) RequestLiveSession(ctx context.Context, _ *empty.Empty) (*pb.LiveSession, error) {

	// TODO(mierdin): need to perform some basic security checks here, like checking to see if this IP registered
	// a bunch of sessions already

	adb := db.NewADMInMem()
	var sessionID string
	i := 0
	for {
		if i > 4 {
			return nil, errors.New("Unable to generate session ID")
		}
		sessionID := db.RandomID(10)
		_, err := adb.GetLiveSession(sessionID)
		if err == nil {
			i++
			continue
		}
		break
	}

	adb.CreateLiveSession(&models.LiveSession{
		ID: sessionID,
		// https://github.com/grpc-ecosystem/grpc-gateway/issues/173
		// SourceIP:   "", // TODO(mierdin): set this once you have it
		Persistent: false,
	})

	return &pb.LiveSession{Id: sessionID}, nil
}
