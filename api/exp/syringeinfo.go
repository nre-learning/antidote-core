package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) GetSyringeInfo(ctx context.Context, _ *empty.Empty) (*pb.SyringeInfo, error) {

	if _, ok := s.buildInfo["buildSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Build SHA not found")
	}

	if _, ok := s.buildInfo["antidoteSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Antidote SHA not found")
	}

	si := pb.SyringeInfo{
		BuildSha:    s.buildInfo["buildSha"],
		AntidoteSha: s.buildInfo["antidoteSha"],
	}

	return &si, nil
}
