package api

import (
	"net"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	labs "github.com/nre-learning/syringe/labs"
	grpc "google.golang.org/grpc"
)

const (
	port = ":50099"
)

func StartAPI(l []*labs.Lab) error {

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterLabsServer(s, &server{labs: l})

	defer s.Stop()
	return s.Serve(lis)

}

type server struct {
	labs []*labs.Lab
}
