package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	swag "github.com/nre-learning/syringe/api/exp/swagger"

	"github.com/nre-learning/syringe/pkg/ui/data/swagger"

	ghandlers "github.com/gorilla/handlers"
	runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
	assetfs "github.com/philips/go-bindata-assetfs"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	gw "github.com/nre-learning/syringe/api/exp/generated"
)

func StartAPI(ls *scheduler.LessonScheduler, grpcPort, httpPort int, buildInfo map[string]string) error {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}

	apiServer := &server{
		liveLessons: make(map[string]*pb.LiveLesson),
		sessions:    make(map[string]map[int32]string),
		scheduler:   ls,
		buildInfo:   buildInfo,
	}

	s := grpc.NewServer()
	pb.RegisterLiveLessonsServiceServer(s, apiServer)
	pb.RegisterLessonDefServiceServer(s, apiServer)
	pb.RegisterSyringeInfoServiceServer(s, apiServer)
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
	err = gw.RegisterSyringeInfoServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", gwmux)
	mux.HandleFunc("/livelesson.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Livelesson))
	})
	mux.HandleFunc("/lessondef.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Lessondef))
	})
	mux.HandleFunc("/syringeinfo.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Syringeinfo))
	})
	serveSwagger(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: grpcHandlerFunc(s, mux),
	}
	go srv.ListenAndServe()

	log.WithFields(log.Fields{
		"gRPC Port": grpcPort,
		"HTTP Port": httpPort,
	}).Info("Syringe API started.")

	// Begin periodically exporting metrics to TSDB
	go apiServer.startTSDBExport()

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

			} else if result.Operation == scheduler.OperationType_GC {
				for i := range result.GCLessons {
					cleanedNs := result.GCLessons[i]
					// 14-6viedvg5rctwdpcc-ns
					lessonId, _ := strconv.ParseInt(strings.Split(cleanedNs, "-")[0], 10, 32)
					sessionId := strings.Split(cleanedNs, "-")[1]

					if _, ok := apiServer.sessions[sessionId]; ok {
						if lessonUuid, ok := apiServer.sessions[sessionId][int32(lessonId)]; ok {

							// Delete UUID from livelessons, and then delete from sessions map
							delete(apiServer.liveLessons, lessonUuid)
							delete(apiServer.sessions[sessionId], int32(lessonId))
						}
					}
				}
			} else {
				log.Error("FOO")
			}
		} else {
			log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
			apiServer.liveLessons[result.Uuid] = &pb.LiveLesson{Error: true}
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

	buildInfo map[string]string
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
	return ghandlers.LoggingHandler(os.Stdout, handlerFunc)
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
