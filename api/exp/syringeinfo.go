package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *SyringeAPIServer) GetSyringeInfo(ctx context.Context, _ *empty.Empty) (*pb.SyringeInfo, error) {

	if _, ok := s.BuildInfo["buildSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Build SHA not found")
	}

	if _, ok := s.BuildInfo["antidoteSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Antidote SHA not found")
	}

	si := pb.SyringeInfo{
		BuildSha:    s.BuildInfo["buildSha"],
		AntidoteSha: s.BuildInfo["antidoteSha"],
	}

	return &si, nil
}
