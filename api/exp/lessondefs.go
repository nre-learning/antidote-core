package api

import (
	"context"

	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) GetLessonDefs(ctx context.Context) ([]*pb.LessonDef, error) {
	return []*pb.LessonDef{}, nil
}
