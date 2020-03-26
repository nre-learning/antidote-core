package api

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/golang/glog"
	nats "github.com/nats-io/nats.go"
	swag "github.com/nre-learning/antidote-core/api/exp/swagger"
	config "github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"

	"github.com/nre-learning/antidote-core/pkg/ui/data/swagger"

	runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	assetfs "github.com/philips/go-bindata-assetfs"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	gw "github.com/nre-learning/antidote-core/api/exp/generated"
)

// AntidoteAPI handles incoming requests from antidote-web, or other gRPC clients
type AntidoteAPI struct {
	Db     db.DataManager
	Config config.AntidoteConfig
	NC     *nats.Conn
	NEC    *nats.EncodedConn

	BuildInfo map[string]string
}

// Start runs the API server. Meant to be executed in a goroutine, as it will block indefinitely
func (apiServer *AntidoteAPI) Start() error {

	grpcPort := apiServer.Config.GRPCPort
	httpPort := apiServer.Config.HTTPPort

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLiveLessonsServiceServer(grpcServer, apiServer)
	pb.RegisterLiveSessionsServiceServer(grpcServer, apiServer)
	pb.RegisterCurriculumServiceServer(grpcServer, apiServer)
	pb.RegisterCollectionServiceServer(grpcServer, apiServer)
	pb.RegisterLessonServiceServer(grpcServer, apiServer)
	pb.RegisterAntidoteInfoServiceServer(grpcServer, apiServer)
	defer grpcServer.Stop()

	// Start grpc server
	go grpcServer.Serve(lis)

	// Start REST proxy
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	// Register GRPC-gateway (HTTP) endpoints
	err = gw.RegisterLiveLessonsServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterLiveSessionsServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterLessonServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterAntidoteInfoServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterCollectionServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterCurriculumServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}

	// Handle swagger requests
	mux := http.NewServeMux()
	mux.Handle("/", gwmux)
	mux.HandleFunc("/livelesson.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Livelesson))
	})
	mux.HandleFunc("/livesession.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Livelesson))
	})
	mux.HandleFunc("/lesson.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Lesson))
	})
	mux.HandleFunc("/antidoteinfo.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Antidoteinfo))
	})
	mux.HandleFunc("/collection.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Collection))
	})
	mux.HandleFunc("/curriculum.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Curriculum))
	})
	serveSwagger(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: grpcHandlerFunc(grpcServer, mux),
	}
	go srv.ListenAndServe()

	log.WithFields(log.Fields{
		"gRPC Port": grpcPort,
		"HTTP Port": httpPort,
	}).Info("Antidote API started.")

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {

	// Add handler for grpc server
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(tamird): point to merged gRPC code rather than a PR.
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})

	// Add gorilla's logging handler for standards-based access logging
	// (disabled for now because we have tracing in place)
	// return ghandlers.LoggingHandler(os.Stdout, allowCORS(handlerFunc))

	// Allow CORS (ONLY IN PREPROD)
	return allowCORS(handlerFunc)

}

// allowCORS allows Cross Origin Resoruce Sharing from any origin.
// Don't do this without consideration in production systems.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

// preflightHandler adds the necessary headers in order to serve
// CORS from any origin using the methods "GET", "HEAD", "POST", "PUT", "DELETE"
// We insist, don't do this without consideration in production systems.
func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	glog.Infof("preflight request for %s", r.URL.Path)
}

func serveSwagger(mux *http.ServeMux) {
	mime.AddExtensionType(".svg", "image/svg+xml")

	// Expose files in third_party/swagger-ui/ on <host>/swagger
	fileServer := http.FileServer(&assetfs.AssetFS{
		Asset:    swagger.Asset,
		AssetDir: swagger.AssetDir,
		Prefix:   "third_party/swagger-ui",
	})
	prefix := "/swagger/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}
