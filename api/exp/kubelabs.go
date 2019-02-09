package api

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) ListKubeLabs(ctx context.Context, _ *empty.Empty) (*pb.KubeLabs, error) {
	pbKl := map[string]*pb.KubeLab{}

	for k, v := range s.scheduler.KubeLabs {
		pbKl[k] = v.ToProtoKubeLab()
	}

	return &pb.KubeLabs{
		Items: pbKl,
	}, nil
}

func (s *server) GetKubeLab(ctx context.Context, uuid *pb.KubeLabUuid) (*pb.KubeLab, error) {
	if _, ok := s.scheduler.KubeLabs[uuid.Id]; !ok {
		return nil, fmt.Errorf("Kubelab %s not found", uuid.Id)
	}
	return s.scheduler.KubeLabs[uuid.Id].ToProtoKubeLab(), nil
}
