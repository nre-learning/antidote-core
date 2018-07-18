package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	labs "github.com/nre-learning/syringe/labs"
	grpc "google.golang.org/grpc"

	gw "github.com/nre-learning/syringe/api/exp/generated"
)

const (
	grpcport = 50099
	httpport = 8085
)

func StartAPI(l []*labs.Lab) error {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcport))
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterLabsServer(s, &server{labs: l})
	defer s.Stop()

	// Start grpc server
	go s.Serve(lis)

	// Start REST proxy
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err = gw.RegisterLabsHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcport), opts)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	// mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
	// 	io.Copy(w, strings.NewReader(pb.Swagger))
	// })

	mux.Handle("/", gwmux)
	// serveSwagger(mux)

	// conn, err := net.Listen("tcp", fmt.Sprintf(":%d", httpport))
	_, err = net.Listen("tcp", fmt.Sprintf(":%d", httpport))
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", httpport),
		Handler: grpcHandlerFunc(s, mux),
		// TLSConfig: &tls.Config{
		// Certificates: []tls.Certificate{*demoKeyPair},
		// NextProtos:   []string{"h2"},
		// },
	}

	log.Infof("grpc on port: %d\n", grpcport)
	log.Infof("http on port: %d\n", httpport)
	// err = srv.Serve(tls.NewListener(conn, srv.TLSConfig))

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return err
	}

	return nil

}

type server struct {
	labs []*labs.Lab
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {

	log.Info("CALLED")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Info("CALLED2")
		// TODO(tamird): point to merged gRPC code rather than a PR.
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

// func serveSwagger(mux *http.ServeMux) {
// 	mime.AddExtensionType(".svg", "image/svg+xml")

// 	// Expose files in third_party/swagger-ui/ on <host>/swagger-ui
// 	fileServer := http.FileServer(&assetfs.AssetFS{
// 		Asset:    swagger.Asset,
// 		AssetDir: swagger.AssetDir,
// 		Prefix:   "third_party/swagger-ui",
// 	})
// 	prefix := "/swagger-ui/"
// 	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
// }
