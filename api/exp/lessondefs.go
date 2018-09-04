package api

import (
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) ListLessonDefs(ctx context.Context, e *empty.Empty) (*pb.LessonDefs, error) {
	return &pb.LessonDefs{}, nil
}

func (s *server) GetLessonDef(ctx context.Context, lid *pb.LessonID) (*pb.LessonDef, error) {

	lessonDef := s.scheduler.LessonDefs[lid.Id]

	log.Debugf("Received request for lesson definition: %v", lessonDef)

	// TODO(mierdin): this little conversion is necessary because the lesson definition structs are not protobufs. Should do this.
	var retLessonDef = pb.LessonDef{
		LessonId: lessonDef.LessonID,
		Stages:   int32(len(lessonDef.Stages)),
	}

	return &retLessonDef, nil
}
