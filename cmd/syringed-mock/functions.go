package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

// LESSON

func (s *MockAPIServer) ListLessons(ctx context.Context, filter *pb.LessonFilter) (*pb.Lessons, error) {

	defs := []*pb.Lesson{}

	return &pb.Lessons{
		Lessons: defs,
	}, nil
}

// var preReqs []int32

func (s *MockAPIServer) GetAllLessonPrereqs(ctx context.Context, lid *pb.LessonID) (*pb.LessonPrereqs, error) {
	return &pb.LessonPrereqs{}, nil
}

func (s *MockAPIServer) GetLesson(ctx context.Context, lid *pb.LessonID) (*pb.Lesson, error) {
	return &pb.Lesson{}, nil
}

// LIVELESSON

func (s *MockAPIServer) RequestLiveLesson(ctx context.Context, lp *pb.LessonParams) (*pb.LessonUUID, error) {
	return &pb.LessonUUID{Id: "abcdef"}, nil
}

func (s *MockAPIServer) GetSyringeState(ctx context.Context, _ *empty.Empty) (*pb.SyringeState, error) {
	return &pb.SyringeState{}, nil
}

func (s *MockAPIServer) HealthCheck(ctx context.Context, _ *empty.Empty) (*pb.HealthCheckMessage, error) {
	return &pb.HealthCheckMessage{}, nil
}

func (s *MockAPIServer) GetLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.LiveLesson, error) {
	return &pb.LiveLesson{}, nil
}

func (s *MockAPIServer) AddSessiontoGCWhitelist(ctx context.Context, session *pb.Session) (*pb.HealthCheckMessage, error) {
	return nil, nil
}

func (s *MockAPIServer) RemoveSessionFromGCWhitelist(ctx context.Context, session *pb.Session) (*pb.HealthCheckMessage, error) {
	return nil, nil
}

func (s *MockAPIServer) GetGCWhitelist(ctx context.Context, _ *empty.Empty) (*pb.Sessions, error) {
	return &pb.Sessions{}, nil
}

func (s *MockAPIServer) ListLiveLessons(ctx context.Context, _ *empty.Empty) (*pb.LiveLessons, error) {
	return &pb.LiveLessons{}, nil
}

func (s *MockAPIServer) KillLiveLesson(ctx context.Context, uuid *pb.LessonUUID) (*pb.KillLiveLessonStatus, error) {
	return &pb.KillLiveLessonStatus{Success: true}, nil
}

func (s *MockAPIServer) RequestVerification(ctx context.Context, uuid *pb.LessonUUID) (*pb.VerificationTaskUUID, error) {
	return &pb.VerificationTaskUUID{Id: "abcdefdfdf"}, nil
}

func (s *MockAPIServer) GetVerification(ctx context.Context, vtUUID *pb.VerificationTaskUUID) (*pb.VerificationTask, error) {
	return &pb.VerificationTask{}, nil
}
