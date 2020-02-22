package api

import (
	"context"

	"github.com/jinzhu/copier"

	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *SyringeAPIServer) ListCollections(ctx context.Context, filter *pb.CollectionFilter) (*pb.Collections, error) {

	collections := []*pb.Collection{}

	for _, c := range s.Scheduler.Curriculum.Collections {
		collections = append(collections, c)
	}

	return &pb.Collections{
		Collections: collections,
	}, nil
}

func (s *SyringeAPIServer) GetCollection(ctx context.Context, filter *pb.CollectionID) (*pb.Collection, error) {

	collection := &pb.Collection{}
	copier.Copy(&collection, s.Scheduler.Curriculum.Collections[filter.Id])

	for lessonID, lesson := range s.Scheduler.Curriculum.Lessons {
		if lesson.Collection == filter.Id {
			collection.Lessons = append(collection.Lessons, &pb.LessonSummary{
				LessonId:          lessonID,
				LessonDescription: lesson.Description,
				LessonName:        lesson.LessonName,
			})
		}
	}

	return collection, nil
}
