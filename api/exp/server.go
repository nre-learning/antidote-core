package api

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
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

// this will keep our state.
type SyringeAPIServer struct {

	// in-memory map of liveLessons, indexed by UUID
	// LiveLesson UUID is a string composed of the lesson ID and the session ID together,
	// separated by a single hyphen. For instance, user session ID 582k2aidfjekxefi and lesson 19
	// will result in 19-582k2aidfjekxefi.
	LiveLessonState map[string]*pb.LiveLesson
	LiveLessonsMu   *sync.Mutex

	// in-memory map of verification tasks, indexed by UUID+stage
	// Similar to livelesson but with stage at the end, e.g. 19-582k2aidfjekxefi-1
	VerificationTasks   map[string]*pb.VerificationTask
	VerificationTasksMu *sync.Mutex

	Scheduler *scheduler.LessonScheduler

	BuildInfo map[string]string
}

func (apiServer *SyringeAPIServer) StartAPI(ls *scheduler.LessonScheduler, buildInfo map[string]string) error {

	grpcPort := apiServer.Scheduler.SyringeConfig.GRPCPort
	httpPort := apiServer.Scheduler.SyringeConfig.HTTPPort

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLiveLessonsServiceServer(grpcServer, apiServer)
	pb.RegisterCurriculumServiceServer(grpcServer, apiServer)
	pb.RegisterLessonServiceServer(grpcServer, apiServer)
	pb.RegisterSyringeInfoServiceServer(grpcServer, apiServer)
	pb.RegisterKubeLabServiceServer(grpcServer, apiServer)
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
	err = gw.RegisterLessonServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}
	err = gw.RegisterSyringeInfoServiceHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf(":%d", grpcPort), opts)
	if err != nil {
		return err
	}

	// Handle swagger requests
	mux := http.NewServeMux()
	mux.Handle("/", gwmux)
	mux.HandleFunc("/livelesson.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Livelesson))
	})
	mux.HandleFunc("/lesson.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Lesson))
	})
	mux.HandleFunc("/syringeinfo.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swag.Syringeinfo))
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
	}).Info("Syringe API started.")

	// Begin periodically exporting metrics to TSDB
	go apiServer.startTSDBExport()

	// Periodic clean-up of verification tasks
	go func() {
		for {
			for id, vt := range apiServer.VerificationTasks {
				if !vt.Working && time.Now().Unix()-vt.Completed.GetSeconds() > 15 {
					apiServer.DeleteVerificationTask(id)
				}
			}
			time.Sleep(time.Second * 5)
		}
	}()

	// Handle results from scheduler asynchronously
	var handlers = map[scheduler.OperationType]interface{}{
		scheduler.OperationType_CREATE: apiServer.handleResultCREATE,
		scheduler.OperationType_DELETE: apiServer.handleResultDELETE,
		scheduler.OperationType_MODIFY: apiServer.handleResultMODIFY,
		scheduler.OperationType_BOOP:   apiServer.handleResultBOOP,
		scheduler.OperationType_VERIFY: apiServer.handleResultVERIFY,
	}
	for {
		result := <-ls.Results

		log.WithFields(log.Fields{
			"Operation": result.Operation,
			"Success":   result.Success,
			"Uuid":      result.Uuid,
		}).Debug("Received result from scheduler.")

		handleFunc := handlers[result.Operation].(func(*scheduler.LessonScheduleResult))
		handleFunc(result)

	}
	return nil
}

func (s *SyringeAPIServer) LiveLessonExists(uuid string) bool {
	_, ok := s.LiveLessonState[uuid]
	return ok
}

func (s *SyringeAPIServer) SetLiveLesson(uuid string, ll *pb.LiveLesson) {
	s.LiveLessonsMu.Lock()
	defer s.LiveLessonsMu.Unlock()

	s.LiveLessonState[uuid] = ll
}

func (s *SyringeAPIServer) UpdateLiveLessonStage(uuid string, stage int32) {
	s.LiveLessonsMu.Lock()
	defer s.LiveLessonsMu.Unlock()

	s.LiveLessonState[uuid].LessonStage = stage
	s.LiveLessonState[uuid].LiveLessonStatus = pb.Status_CONFIGURATION
}

func (s *SyringeAPIServer) DeleteLiveLesson(uuid string) {
	if _, ok := s.LiveLessonState[uuid]; !ok {
		// Nothing to do
		log.Debug("DeleteLiveLesson - Returning early.")
		return
	}
	s.LiveLessonsMu.Lock()
	defer s.LiveLessonsMu.Unlock()
	log.Debugf("DeleteLiveLesson - About to Delete. Current state: %s", s.LiveLessonState)
	delete(s.LiveLessonState, uuid)
	log.Debugf("DeleteLiveLesson - FINISHED. Current state: %s", s.LiveLessonState)
}

func (s *SyringeAPIServer) SetVerificationTask(uuid string, vt *pb.VerificationTask) {
	s.VerificationTasksMu.Lock()
	defer s.VerificationTasksMu.Unlock()

	s.VerificationTasks[uuid] = vt
}

func (s *SyringeAPIServer) DeleteVerificationTask(uuid string) {
	if _, ok := s.VerificationTasks[uuid]; !ok {
		// Nothing to do
		return
	}
	s.VerificationTasksMu.Lock()
	defer s.VerificationTasksMu.Unlock()
	delete(s.VerificationTasks, uuid)
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

	// Allow CORS (ONLY IN PREPROD)
	// Add gorilla's logging handler for standards-based access logging
	return ghandlers.LoggingHandler(os.Stdout, allowCORS(handlerFunc))
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
