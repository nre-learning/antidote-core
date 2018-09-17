package api

import (
	"context"
	"errors"
	"sort"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	def "github.com/nre-learning/syringe/def"
)

func (s *server) ListLessonDefs(ctx context.Context, ldFilter *pb.LessonDefFilter) (*pb.LessonDefs, error) {

	// TODO(mierdin): Convert to more generic nil filter check (we may want to filter via something else later)
	if ldFilter.Category == "" {
		return &pb.LessonDefs{}, errors.New("Must provide category")
	}

	// TODO(mierdin): Okay for now, but not super effecient. Should store in category keys when loaded.
	var retDefs []*pb.LessonDef
	for lessonId, lessonDef := range s.scheduler.LessonDefs {
		log.Debugf("Lesson %d is in the %s category", lessonId, lessonDef.Category)
		if lessonDef.Category == ldFilter.Category {
			retDefs = append(retDefs,
				// TODO(mierdin): this little conversion is necessary because the lesson definition structs are not protobufs. Should do this.
				&pb.LessonDef{
					LessonName: lessonDef.LessonName,
					LessonId:   lessonDef.LessonID,
					Stages:     convertStages(lessonDef.Stages),
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
		Stages:   convertStages(lessonDef.Stages),
	}

	return &retLessonDef, nil
}

// Helper function because lesson definitions aren't protobufs
func convertStages(stages map[int32]*def.LessonStage) []*pb.LessonStage {
	retStages := []*pb.LessonStage{}

	stageIds := []int{}
	for k := range stages {
		stageIds = append(stageIds, int(k))
	}
	sort.Ints(stageIds)

	for i := range stageIds {
		stage := pb.LessonStage{
			StageId:     int32(stageIds[i]),
			Description: stages[int32(stageIds[i])].Description,
		}
		retStages = append(retStages, &stage)
	}

	return retStages
}
