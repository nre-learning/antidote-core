package api

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *SyringeAPIServer) RequestLiveSession(ctx context.Context, _ *empty.Empty) (*pb.LessonUUID, error) {

	// TODO(mierdin): need to perform some basic security checks here, like checking to see if this IP registered
	// a bunch of sessions already

	return &pb.LessonUUID{Id: llID}, nil
}
