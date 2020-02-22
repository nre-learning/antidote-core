package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

// GetSyringeInfo provides detailed information about which version of Syringe, and other related
// software/assets are loaded. Primarily used for a debug banner in the web front-end
func (s *SyringeAPIServer) GetSyringeInfo(ctx context.Context, _ *empty.Empty) (*pb.SyringeInfo, error) {

	if _, ok := s.BuildInfo["buildSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Build SHA not found")
	}

	if _, ok := s.BuildInfo["antidoteSha"]; !ok {
		return &pb.SyringeInfo{}, errors.New("Antidote SHA not found")
	}

	si := pb.SyringeInfo{
		BuildSha:     s.BuildInfo["buildSha"],
		AntidoteSha:  s.BuildInfo["antidoteSha"],
		ImageVersion: s.BuildInfo["imageVersion"],
	}

	return &si, nil
}
