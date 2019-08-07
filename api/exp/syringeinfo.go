package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *SyringeAPIServer) GetSyringeInfo(ctx context.Context, _ *empty.Empty) (*pb.SyringeInfo, error) {

	if _, ok := s.Scheduler.BuildInfo["buildSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Build SHA not found")
	}

	if _, ok := s.Scheduler.BuildInfo["antidoteSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Antidote SHA not found")
	}

	si := pb.SyringeInfo{
		BuildSha:     s.Scheduler.BuildInfo["buildSha"],
		AntidoteSha:  s.Scheduler.BuildInfo["antidoteSha"],
		ImageVersion: s.Scheduler.BuildInfo["imageVersion"],
	}

	return &si, nil
}
