package api

import (
	"context"
	"errors"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	def "github.com/nre-learning/syringe/def"
)

func (s *server) ListLessonDefs(ctx context.Context, _ *empty.Empty) (*pb.LessonCategoryMap, error) {

	retMap := map[string]*pb.LessonDefs{}

	// TODO(mierdin): Okay for now, but not super efficient. Should store in category keys when loaded.
	for _, lessonDef := range s.scheduler.LessonDefs {

		// Initialize category
		if _, ok := retMap[lessonDef.Category]; !ok {
			retMap[lessonDef.Category] = &pb.LessonDefs{
				LessonDefs: []*pb.LessonDef{},
			}
		}

		retMap[lessonDef.Category].LessonDefs = append(retMap[lessonDef.Category].LessonDefs,
			// TODO(mierdin): this little conversion is necessary because the lesson definition structs are not protobufs. Should do this.
			&pb.LessonDef{
				LessonName: lessonDef.LessonName,
				LessonId:   lessonDef.LessonID,
				Stages:     convertStages(lessonDef.Stages),
			},
		)
	}

	return &pb.LessonCategoryMap{
		LessonCategories: retMap,
	}, nil
}

func (s *server) GetLessonDef(ctx context.Context, lid *pb.LessonID) (*pb.LessonDef, error) {

	if _, ok := s.scheduler.LessonDefs[lid.Id]; !ok {
		return nil, errors.New("Invalid lesson ID")
	}

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
