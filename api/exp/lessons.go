package api

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	models "github.com/nre-learning/syringe/db/models"
)

func (s *SyringeAPIServer) ListLessons(ctx context.Context, filter *pb.LessonFilter) (*pb.Lessons, error) {

	defs := []*pb.Lesson{}

	// TODO(mierdin): Okay for now, but not super efficient. Should store in category keys when loaded.
	for _, lesson := range s.Curriculum.Lessons {

		if filter.Category == "" {
			defs = append(defs, lesson)
			continue
		}

		if lesson.Category == filter.Category {
			defs = append(defs, lesson)
		}
	}

	return &pb.Lessons{
		Lessons: defs,
	}, nil
}

// var preReqs []int32

func (s *SyringeAPIServer) GetAllLessonPrereqs(ctx context.Context, lid *pb.LessonID) (*pb.LessonPrereqs, error) {

	// Preload the requested lesson ID so we can strip it before returning
	pr := s.getPrereqs(lid.Id, []int32{lid.Id})
	log.Debugf("Getting prerequisites for Lesson %d: %d", lid.Id, pr)

	return &pb.LessonPrereqs{
		// Remove first item from slice - this is the lesson ID being requested
		Prereqs: pr[1:],
	}, nil
}

func (s *SyringeAPIServer) getPrereqs(lessonID int32, currentPrereqs []int32) []int32 {

	// Return if lesson ID doesn't exist
	if _, ok := s.Curriculum.Lessons[lessonID]; !ok {
		return currentPrereqs
	}

	// Add this lessonID to prereqs if doesn't already exist
	if !isAlreadyInSlice(lessonID, currentPrereqs) {
		currentPrereqs = append(currentPrereqs, lessonID)
	}

	// Return if lesson doesn't have prerequisites
	lesson := s.Curriculum.Lessons[lessonID]
	if len(lesson.Prereqs) == 0 {
		return currentPrereqs
	}

	// Call recursion for lesson IDs that need it
	for i := range lesson.Prereqs {
		pid := lesson.Prereqs[i]
		currentPrereqs = s.getPrereqs(pid, currentPrereqs)
	}

	return currentPrereqs
}

func isAlreadyInSlice(lessonID int32, currentPrereqs []int32) bool {
	for i := range currentPrereqs {
		if currentPrereqs[i] == lessonID {
			return true
		}
	}
	return false
}

func (s *SyringeAPIServer) GetLesson(ctx context.Context, lid *pb.LessonID) (*pb.Lesson, error) {

	if _, ok := s.Curriculum.Lessons[lid.Id]; !ok {
		return nil, errors.New("Invalid lesson ID")
	}

	lesson := s.Curriculum.Lessons[lid.Id]

	return lesson, nil
}

// lessonDBToAPI translates a single Lesson from the `db` package models into the
// api package's equivalent
func lessonDBToAPI(dbLesson *models.Lesson) (*pb.Lesson, error) {
	lessonAPI := *pb.Lesson{}
	return lessonAPI, nil
}

// lessonAPIToDB translates a single Lesson from the `api` package models into the
// `db` package's equivalent
func lessonAPIToDB(pbLesson *pb.Lesson) (*models.Lesson, error) {
	lessonAPI := *pb.Lesson{}
	return lessonAPI, nil
}
