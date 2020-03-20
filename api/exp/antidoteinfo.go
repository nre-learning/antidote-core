package api

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"
)

// GetAntidoteInfo provides detailed information about which version of Antidote, and other related
// software/assets are loaded. Primarily used for a debug banner in the web front-end
func (s *AntidoteAPI) GetAntidoteInfo(ctx context.Context, _ *empty.Empty) (*pb.AntidoteInfo, error) {

	if _, ok := s.BuildInfo["buildSha"]; !ok {
		return &pb.AntidoteInfo{}, errors.New("Build SHA not found")
	}

	if _, ok := s.BuildInfo["buildVersion"]; !ok {
		return &pb.AntidoteInfo{}, errors.New("Build Version not found")
	}

	if _, ok := s.BuildInfo["curriculumVersion"]; !ok {
		return &pb.AntidoteInfo{}, errors.New("Curriculum Version not found")
	}

	ai := pb.AntidoteInfo{
		BuildSha:          s.BuildInfo["buildSha"],
		BuildVersion:      s.BuildInfo["buildVersion"],
		CurriculumVersion: s.BuildInfo["curriculumVersion"],
	}

	return &ai, nil
}
