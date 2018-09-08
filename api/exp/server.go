package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
	grpc "google.golang.org/grpc"

	gw "github.com/nre-learning/syringe/api/exp/generated"
)

func StartAPI(ls *scheduler.LessonScheduler, grpcPort, httpPort int) error {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}

	apiServer := &server{
		liveLessons: make(map[string]*pb.LiveLesson),
		sessions:    make(map[string]map[int32]string),
		scheduler:   ls,
	}

	// go func() {
	// 	for {
	// 		time.Sleep(1 * time.Second)
	// 		log.Warn(apiServer.liveLabs)
	// 	}
	// }()

	s := grpc.NewServer()
	pb.RegisterLiveLessonsServiceServer(s, apiServer)
	pb.RegisterLessonDefServiceServer(s, apiServer)
	defer s.Stop()

	// Start grpc server
	go s.Serve(lis)

	// Start REST proxy
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err = gw.RegisterLiveLessonsServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterLessonDefServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
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
	// _, err = net.Listen("tcp", fmt.Sprintf(":%d", httpport))
	// if err != nil {
	// 	panic(err)
	// }

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: grpcHandlerFunc(s, mux),
		// TLSConfig: &tls.Config{
		// Certificates: []tls.Certificate{*demoKeyPair},
		// NextProtos:   []string{"h2"},
		// },
	}

	log.Debugf("gRPC server listening on port: %d\n", grpcPort)
	log.Debugf("HTTP gateway listening on port: %d\n", httpPort)
	log.Debug("Started.")
	// err = srv.Serve(tls.NewListener(conn, srv.TLSConfig))

	go srv.ListenAndServe()
	// if err != nil {
	// 	log.Fatal("ListenAndServe: ", err)
	// 	return err
	// }

	for {
		result := <-ls.Results

		log.Debugf("Received result message: %v", result)

		if result.Success {
			if result.Operation == scheduler.OperationType_CREATE {

				log.Debugf("Setting liveLesson %s: %v", result.Uuid, result.KubeLab.ToLiveLesson())
				apiServer.liveLessons[result.Uuid] = result.KubeLab.ToLiveLesson()
			} else if result.Operation == scheduler.OperationType_DELETE {
				delete(apiServer.liveLessons, result.Uuid)
			} else if result.Operation == scheduler.OperationType_MODIFY {
				log.Debugf("Setting liveLesson %s: %v", result.Uuid, result.KubeLab.ToLiveLesson())
				apiServer.liveLessons[result.Uuid] = result.KubeLab.ToLiveLesson()
			} else {
				log.Error("FOO")
			}
		} else {
			log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
		}
	}

	return nil

}

// this will keep our state.
type server struct {

	// in-memory map of liveLessons, indexed by UUID
	liveLessons map[string]*pb.LiveLesson

	scheduler *scheduler.LessonScheduler

	// map of session IDs maps containing lesson ID and corresponding lesson UUID
	sessions map[string]map[int32]string
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

var validShortID = regexp.MustCompile("^[a-z0-9]{12}$")

// IsShortID determine if an arbitrary string *looks like* a short ID.
func IsShortID(id string) bool {
	return validShortID.MatchString(id)
}

// TruncateID returns a shorthand version of a string identifier for convenience.
// A collision with other shorthands is very unlikely, but possible.
// In case of a collision a lookup with TruncIndex.Get() will fail, and the caller
// will need to use a langer prefix, or the full-length Id.
func TruncateID(id string) string {
	trimTo := 12
	if len(id) < trimTo {
		trimTo = len(id)
	}
	return id[:trimTo]
}

// GenerateUUID returns an unique id
func GenerateUUID() string {
	for {
		id := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, id); err != nil {
			panic(err) // This shouldn't happen
		}
		value := hex.EncodeToString(id)
		// if we try to parse the truncated for as an int and we don't have
		// an error then the value is all numberic and causes issues when
		// used as a hostname. ref #3869
		if _, err := strconv.ParseInt(TruncateID(value), 10, 64); err == nil {
			continue
		}
		return value
	}
}
