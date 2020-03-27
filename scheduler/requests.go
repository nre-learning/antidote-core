package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	corev1 "k8s.io/api/core/v1"

	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
)

func (s *AntidoteScheduler) handleRequestCREATE(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_create", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, newRequest.LiveLessonID)

	span.LogEvent(fmt.Sprintf("Generated namespace name %s", nsName))

	ll, err := s.Db.GetLiveLesson(span.Context(), newRequest.LiveLessonID)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = s.createK8sStuff(span.Context(), newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = s.waitUntilReachable(span.Context(), ll)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = s.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_CONFIGURATION)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = s.configureStuff(span.Context(), nsName, ll, newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	// Set network policy ONLY after configuration has had a chance to take place. Once this is in place,
	// only config pods spawned by Jobs will have internet access, so if this takes place earlier, lessons
	// won't initially come up at all.
	if s.Config.AllowEgress {
		s.createNetworkPolicy(span.Context(), nsName)
	}

	_ = s.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)

	// Inject span context and send LSR into NATS
	tracer := ot.GlobalTracer()
	var t services.TraceMsg
	if err := tracer.Inject(span.Context(), ot.Binary, &t); err != nil {
		// log.Fatalf("%v for Inject.", err)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
	reqBytes, _ := json.Marshal(newRequest)
	t.Write(reqBytes)
	s.NC.Publish("antidote.lsr.completed", t.Bytes())
}

func (s *AntidoteScheduler) handleRequestMODIFY(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_modify", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, newRequest.LiveLessonID)

	ll, err := s.Db.GetLiveLesson(span.Context(), newRequest.LiveLessonID)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = s.configureStuff(span.Context(), nsName, ll, newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = s.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}

	err = s.boopNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
}

func (s *AntidoteScheduler) handleRequestBOOP(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_boop", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, newRequest.LiveLessonID)

	err := s.boopNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
}

// handleRequestDELETE handles a livelesson deletion request by first sending a delete request
// for the corresponding namespace, and then cleaning up local state.
func (s *AntidoteScheduler) handleRequestDELETE(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_delete", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(s.Config.InstanceID, newRequest.LiveLessonID)
	err := s.deleteNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}
	_ = s.Db.DeleteLiveLesson(span.Context(), newRequest.LiveLessonID)
}

// createK8sStuff is a high-level workflow for creating all of the things necessary for a new instance
// of a livelesson. Pods, services, networks, networkpolicies, ingresses, etc to support a new running
// lesson are all created as part of this workflow.
func (s *AntidoteScheduler) createK8sStuff(sc ot.SpanContext, req services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_k8s_create_stuff", ot.ChildOf(sc))
	defer span.Finish()

	ns, err := s.createNamespace(span.Context(), req)
	if err != nil {
		log.Error(err)
	}

	_ = s.syncSecret(span.Context(), ns.ObjectMeta.Name)
	// if err != nil {
	// 	log.Errorf("Unable to sync secret into this namespace. Ingress-based resources (like http presentations or jupyter notebooks) may not work: %v", err)
	// }

	lesson, err := s.Db.GetLesson(span.Context(), req.LessonSlug)
	if err != nil {
		return err
	}

	ll, err := s.Db.GetLiveLesson(span.Context(), req.LiveLessonID)
	if err != nil {
		return err
	}

	// Append endpoint and create ingress for jupyter lab guide if necessary
	if usesJupyterLabGuide(lesson) {

		jupyterEp := &models.LiveEndpoint{
			Name:  "jupyterlabguide",
			Image: fmt.Sprintf("antidotelabs/jupyter:%s", s.BuildInfo["imageVersion"]),
			Ports: []int32{8888},
		}
		ll.LiveEndpoints[jupyterEp.Name] = jupyterEp

		_, err := s.createIngress(
			span.Context(),
			ns.ObjectMeta.Name,
			jupyterEp,
			&models.LivePresentation{
				Name: "web",
				Port: 8888,
			},
		)
		if err != nil {
			return fmt.Errorf("Unable to create ingress resource - %v", err)
		}
	}

	// Create networks from connections property
	for c := range lesson.Connections {
		connection := lesson.Connections[c]
		_, err := s.createNetwork(span.Context(), c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), req)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	createdPods := make(map[string]*corev1.Pod)

	// Create pods and services
	for d := range ll.LiveEndpoints {
		ep := ll.LiveEndpoints[d]

		// createPod doesn't try to ensure a certain pod status. That's done later
		newPod, err := s.createPod(span.Context(),
			ep,
			getMemberNetworks(ep.Name, lesson.Connections),
			req,
		)
		if err != nil {
			log.Error(err)
			return err
		}

		createdPods[newPod.ObjectMeta.Name] = newPod

		// Expose via service if needed
		if len(newPod.Spec.Containers[0].Ports) > 0 {
			svc, err := s.createService(
				span.Context(),
				newPod,
				req,
			)
			if err != nil {
				log.Error(err)
				return err
			}

			// Update livelesson liveendpoint with cluster IP
			s.Db.UpdateLiveLessonEndpointIP(span.Context(), ll.ID, ep.Name, svc.Spec.ClusterIP)
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
				return err
			}

		}

		// Create ingresses for http presentations
		for pr := range ep.Presentations {
			p := ep.Presentations[pr]

			if p.Type == "http" {
				_, err := s.createIngress(
					span.Context(),
					ns.ObjectMeta.Name,
					ep,
					p,
				)
				if err != nil {
					log.Error(err)
					return err
				}
			}
		}
	}

	err = s.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_BOOTING)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	// Before moving forward with network-based health checks, let's look back at the pods
	// we've deployed, and wait until they're in a "Running" status. This allows us to keep a hold
	// of maximum amounts of context for troubleshooting while we have it
	wg := new(sync.WaitGroup)
	wg.Add(len(createdPods))

	failLesson := false

	podStatusSpan := ot.StartSpan("scheduler_pod_status", ot.ChildOf(span.Context()))
	for name, pod := range createdPods {
		go func(podStatusSpan ot.Span, name string, pod *corev1.Pod) {

			defer wg.Done()

			for i := 0; i < 300; i++ {

				rdy, err := s.getPodStatus(podStatusSpan, pod)
				if err != nil {
					failLesson = true
					return
				}

				if rdy {
					delete(createdPods, name)
					return
				}

				time.Sleep(1 * time.Second)
			}

			failedLogs := s.getPodLogs(pod)
			span.LogEventWithPayload("podFailureLogs", services.SafePayload(failedLogs))

			err = fmt.Errorf("Timed out waiting for %s to start", name)
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			failLesson = true
			return
		}(podStatusSpan, name, pod)
	}

	wg.Wait()
	podStatusSpan.Finish()

	// At this point, the only pods left in createdPods should be ones that failed to ready
	if failLesson || len(createdPods) > 0 {

		failedPodNames := []string{}
		for k := range createdPods {
			failedPodNames = append(failedPodNames, k)
		}

		span.LogFields(log.Object("failedPodNames", failedPodNames))
		ext.Error.Set(span, true)
		return errors.New("Some pods failed to start")
	}

	return nil
}
