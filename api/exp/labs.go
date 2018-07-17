package api

import (
	"context"
	"strconv"
	// log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) RequestLab(ctx context.Context, newNode *pb.LabParams) (*pb.LabUUID, error) {
	return &pb.LabUUID{Id: 1}, nil
}

func (s *server) GetLab(ctx context.Context, uuid *pb.LabUUID) (*pb.Lab, error) {

	port1, _ := strconv.Atoi(s.labs[0].LabConnections["csrx1"])

	labMap := map[int]*pb.Lab{
		1: &pb.Lab{
			LabUUID: 1,
			LabId:   1,
			Devices: []*pb.LabDevice{
				{
					Name: "crx01",
					Port: int32(port1),
				},
			},
			Ready: true,
		},
	}

	return labMap[int(uuid.Id)], nil
}
