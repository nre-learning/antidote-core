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

	"github.com/golang/protobuf/ptypes/timestamp"
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
		liveLessonState:     make(map[string]*pb.LiveLesson),
		liveLessonsMu:       &sync.Mutex{},
		verificationTasks:   make(map[string]*pb.VerificationTask),
		verificationTasksMu: &sync.Mutex{},
		scheduler:           ls,
		buildInfo:           buildInfo,
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLiveLessonsServiceServer(grpcServer, apiServer)
	pb.RegisterLessonDefServiceServer(grpcServer, apiServer)
	pb.RegisterSyringeInfoServiceServer(grpcServer, apiServer)
	defer grpcServer.Stop()

	// Start grpc server
	go grpcServer.Serve(lis)

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
			log.Debugf("Verification Tasks: %s", apiServer.verificationTasks)
			for id, vt := range apiServer.verificationTasks {
				if !vt.Working && time.Now().Unix()-vt.Completed.GetSeconds() > 30 {
					apiServer.DeleteVerificationTask(id)
				}
			}
			time.Sleep(time.Second * 5)
		}
	}()

	for {
		result := <-ls.Results

		log.WithFields(log.Fields{
			"Operation": result.Operation,
			"Success":   result.Success,
			"Uuid":      result.Uuid,
		}).Debug("Received result from scheduler.")

		if result.Success {

			// Scheduler operation successful - just need to update the state in memory accordingly
			if result.Operation == scheduler.OperationType_CREATE {
				apiServer.recordProvisioningTime(result.ProvisioningTime, result)
				apiServer.SetLiveLesson(result.Uuid, result.KubeLab.ToLiveLesson())
			} else if result.Operation == scheduler.OperationType_MODIFY {
				apiServer.SetLiveLesson(result.Uuid, result.KubeLab.ToLiveLesson())
			} else if result.Operation == scheduler.OperationType_GC {
				for i := range result.GCLessons {

					// TODO(mierdin): why am I doing this? The other functions don't strip this
					uuid := strings.TrimRight(result.GCLessons[i], "-ns")
					apiServer.DeleteLiveLesson(uuid)
				}
			} else if result.Operation == scheduler.OperationType_VERIFY {
				vtUUID := fmt.Sprintf("%s-%d", result.Uuid, result.Stage)

				vt := apiServer.verificationTasks[vtUUID]
				vt.Working = false
				vt.Success = result.Success
				if result.Success == true {
					vt.Message = "Successfully verified"
				} else {

					// TODO(mierdin): Provide an optional field for the author to provide a hint that overrides this.
					vt.Message = "Failed to verify"
				}
				vt.Completed = &timestamp.Timestamp{
					Seconds: time.Now().Unix(),
				}

				apiServer.SetVerificationTask(vtUUID, vt)
			} else {
				log.Error("FOO")
			}
		} else {
			log.Errorf("Problem encountered in request %s: %s", result.Uuid, result.Message)
			apiServer.SetLiveLesson(result.Uuid, &pb.LiveLesson{Error: true})
		}
	}

	return nil

}

// this will keep our state.
type server struct {

	// in-memory map of liveLessons, indexed by UUID
	// LiveLesson UUID is a string composed of the lesson ID and the session ID together,
	// separated by a single hyphen. For instance, user session ID 582k2aidfjekxefi and lesson 19
	// will result in 19-582k2aidfjekxefi.
	liveLessonState map[string]*pb.LiveLesson
	liveLessonsMu   *sync.Mutex

	// in-memory map of verification tasks, indexed by UUID+stage
	// Similar to livelesson but with stage at the end, e.g. 19-582k2aidfjekxefi-1
	verificationTasks   map[string]*pb.VerificationTask
	verificationTasksMu *sync.Mutex

	scheduler *scheduler.LessonScheduler

	buildInfo map[string]string
}

func (s *server) LiveLessonExists(uuid string) bool {
	_, ok := s.liveLessonState[uuid]
	return ok
}

func (s *server) SetLiveLesson(uuid string, ll *pb.LiveLesson) {
	s.liveLessonsMu.Lock()
	defer s.liveLessonsMu.Unlock()

	s.liveLessonState[uuid] = ll
}

func (s *server) UpdateLiveLessonStage(uuid string, stage int32) {
	s.liveLessonsMu.Lock()
	defer s.liveLessonsMu.Unlock()

	s.liveLessonState[uuid].LessonStage = stage
	s.liveLessonState[uuid].LiveLessonStatus = pb.Status_CONFIGURATION
}

func (s *server) DeleteLiveLesson(uuid string) {
	if _, ok := s.liveLessonState[uuid]; !ok {
		// Nothing to do
		return
	}
	s.liveLessonsMu.Lock()
	defer s.liveLessonsMu.Unlock()
	delete(s.liveLessonState, uuid)
}

func (s *server) SetVerificationTask(uuid string, vt *pb.VerificationTask) {
	s.verificationTasksMu.Lock()
	defer s.verificationTasksMu.Unlock()

	s.verificationTasks[uuid] = vt
}

func (s *server) DeleteVerificationTask(uuid string) {
	if _, ok := s.verificationTasks[uuid]; !ok {
		// Nothing to do
		return
	}
	s.verificationTasksMu.Lock()
	defer s.verificationTasksMu.Unlock()
	delete(s.verificationTasks, uuid)
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
