package api

import (
	"context"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
)

// ImportCurriculum provides a single function for anything within Syringe to load a Curriculum
// into memory. It goes through the logic of importing and validating everything within a curriculum,
// including lessons, collections, etc. This allows both syrctl and the syringe scheduler to simply
// load things exactly the same way every time.
func ImportCurriculum(config *config.SyringeConfig) (*pb.Curriculum, error) {

	curriculum := &pb.Curriculum{}

	// Load lessons
	lessons, err := ImportLessons(config)
	if err != nil {
		log.Warn(err)
	}
	curriculum.Lessons = lessons

	return curriculum, nil
}

// GetCurriculumInfo is designed to only get high-level information about the loaded Curriculum. Specifics on lessons, collections, and more
// are given their own first-level API endpoint elsewhere, but will be pulled from the loaded Curriculum struct being described here.
func (s *SyringeAPIServer) GetCurriculumInfo(ctx context.Context, filter *pb.CurriculumFilter) (*pb.CurriculumInfo, error) {
	return &pb.CurriculumInfo{
		Name:        s.Scheduler.Curriculum.Name,
		Description: s.Scheduler.Curriculum.Description,
		Website:     s.Scheduler.Curriculum.Website,
	}, nil
}
