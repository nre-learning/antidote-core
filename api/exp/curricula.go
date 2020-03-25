package api

import (
	"context"
	"errors"

	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	log "github.com/sirupsen/logrus"
)

// GetCurriculumInfo is designed to only get high-level information about the loaded Curriculum. Specifics on lessons, collections, and more
// are given their own first-level API endpoint elsewhere, but will be pulled from the loaded Curriculum struct being described here.
func (s *AntidoteAPI) GetCurriculumInfo(ctx context.Context, filter *pb.CurriculumFilter) (*pb.CurriculumInfo, error) {

	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan("api_curriculum_getinfo", ext.SpanKindRPCClient)
	defer span.Finish()

	curriculum, err := s.Db.GetCurriculum(span.Context())
	if err != nil {
		log.Error(err)
		return nil, errors.New("Problem retrieving curriculum details")
	}

	return &pb.CurriculumInfo{
		Name:        curriculum.Slug,
		Description: curriculum.Description,
		Website:     curriculum.Website,
	}, nil
}
