package api

import (
	"context"
	"errors"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) ListLessonDefs(ctx context.Context, ldFilter *pb.LessonDefFilter) (*pb.LessonDefs, error) {

	// TODO(mierdin): Convert to more generic nil filter check (we may want to filter via something else later)
	if ldFilter.Category == "" {
		return &pb.LessonDefs{}, errors.New("Must provide category")
	}

	// TODO(mierdin): Okay for now, but not super effecient. Should store in category keys when loaded.
	var retDefs []*pb.LessonDef
	log.Debugf("Looking for all %s lessons", ldFilter.Category)
	log.Debugf("ldFilter - %s", ldFilter)
	for lessonId, lessonDef := range s.scheduler.LessonDefs {
		log.Debugf("%d is in the %s category", lessonId, lessonDef.Category)
		if lessonDef.Category == ldFilter.Category {
			retDefs = append(retDefs,
				// TODO(mierdin): this little conversion is necessary because the lesson definition structs are not protobufs. Should do this.
				&pb.LessonDef{
					LessonName: lessonDef.LessonName,
					LessonId:   lessonDef.LessonID,
					Stages:     int32(len(lessonDef.Stages)),
				},
			)
		}
	}

	return &pb.LessonDefs{
		Lessondefs: retDefs,
	}, nil
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
