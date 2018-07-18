package api

import (
	"context"
	"errors"
	"strconv"
	// log "github.com/Sirupsen/logrus"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func (s *server) RequestLab(ctx context.Context, newNode *pb.LabParams) (*pb.LabUUID, error) {
	return &pb.LabUUID{Id: 1}, nil
}

func (s *server) GetLab(ctx context.Context, uuid *pb.LabUUID) (*pb.Lab, error) {
	log.Info("GOT HERE 4")
	port1, _ := strconv.Atoi(s.labs[0].LabConnections["csrx1"])

	if uuid.Id == 0 {
		msg := "Lab UUID cannot be nil or 0"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	log.Info(uuid.Id)

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

	log.Infof("About to return %s", labMap[int(uuid.Id)])

	return labMap[int(uuid.Id)], nil
}
