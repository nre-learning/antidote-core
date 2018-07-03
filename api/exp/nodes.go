package api

// pb "github.com/nre-learning/syringe/api/exp/generated"

// // CreateNode creates a new CreateNode
// func (s *server) CreateNode(ctx context.Context, newNode *pb.Node) (*pb.NodeResponse, error) {
// 	s.nodes[newNode.Id] = newNode
// 	return &pb.NodeResponse{Id: newNode.Id, Success: true}, nil
// }

// // GetNode returns a single node
// func (s *server) GetNode(ctx context.Context, f *pb.NodeRequest) (*pb.Node, error) {
// 	return s.nodes[f.Id], nil
// }

// // ListNodes returns a list of Nodes
// func (s *server) ListNodes(ctx context.Context, f *pb.NodeFilter) (*pb.NodeList, error) {
// 	theseNodes := []*pb.Node{}
// 	for _, v := range s.nodes {
// 		theseNodes = append(theseNodes, v)
// 	}
// 	return &pb.NodeList{Nodes: theseNodes}, nil
// }
