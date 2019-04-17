package api

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *SyringeAPIServer) ListKubeLabs(ctx context.Context, _ *empty.Empty) (*pb.KubeLabs, error) {
	pbKl := map[string]*pb.KubeLab{}

	for k, v := range s.Scheduler.KubeLabs {
		pbKl[k] = v.ToProtoKubeLab()
	}

	return &pb.KubeLabs{
		Items: pbKl,
	}, nil
}

func (s *SyringeAPIServer) GetKubeLab(ctx context.Context, uuid *pb.KubeLabUuid) (*pb.KubeLab, error) {
	if _, ok := s.Scheduler.KubeLabs[uuid.Id]; !ok {
		return nil, fmt.Errorf("Kubelab %s not found", uuid.Id)
	}
	return s.Scheduler.KubeLabs[uuid.Id].ToProtoKubeLab(), nil
}
