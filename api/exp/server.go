package api

// pb "github.com/nre-learning/syringe/api/exp/generated"

// const (
// 	port = ":50099"
// )

// func StartAPI() error {

// 	lis, err := net.Listen("tcp", port)
// 	if err != nil {
// 		log.Errorf("failed to listen: %v", err)
// 	}
// 	// Creates a new gRPC server
// 	s := grpc.NewServer()
// 	pb.RegisterNodesServer(s, &server{nodes: map[int32]*pb.Node{}})

// 	defer s.Stop()
// 	return s.Serve(lis)

// }

// type server struct {

// 	// storing data in memory for now
// 	nodes map[int32]*pb.Node
// }
