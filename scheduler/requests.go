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
		_ = s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = s.waitUntilReachable(span.Context(), ll)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = s.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_CONFIGURATION)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = s.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
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
	if !s.Config.AllowEgress {
		s.createNetworkPolicy(span.Context(), nsName)
	}

	_ = s.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)

	// Inject span context and send LSR into NATS
	tracer := ot.GlobalTracer()
	var t services.TraceMsg
	if err := tracer.Inject(span.Context(), ot.Binary, &t); err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
	reqBytes, _ := json.Marshal(newRequest)
	t.Write(reqBytes)
	s.NC.Publish(services.LsrCompleted, t.Bytes())
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
	}

	// Make sure to not return earlier than this, so we make sure the state is cleaned up
	// no matter what
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

	// Sync TLS certificate into the lesson namespace (and optionally, docker pull credentials)
	_ = s.syncSecret(span.Context(), s.Config.SecretsNamespace, ns.ObjectMeta.Name, s.Config.TLSCertName)
	if s.Config.PullCredName != "" {
		_ = s.syncSecret(span.Context(), s.Config.SecretsNamespace, ns.ObjectMeta.Name, s.Config.PullCredName)
	}

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

		nsName := generateNamespaceName(s.Config.InstanceID, req.LiveLessonID)

		_, err := s.createIngress(
			span.Context(),
			ns.ObjectMeta.Name,
			jupyterEp,
			&models.LivePresentation{
				Name:      "web",
				Port:      8888,
				HepDomain: fmt.Sprintf("%s-jupyterlabguide-web.%s", nsName, s.Config.HEPSDomain),
			},
		)
		if err != nil {
			return fmt.Errorf("Unable to create ingress resource - %v", err)
		}
	}

	// Create networks from connections property
	for c := range lesson.Connections {
		connection := lesson.Connections[c]
		_, err := s.createNetwork(span.Context(), c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), connection.Subnet, req)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	createdPods := make(map[string]*corev1.Pod)

	// The LiveEndpoints field of the LiveLesson model is a map, not a slice. This means that the original order from the lesson definition
	// is lost when the API's initializeLiveEndpoints function populates this from the original slice into the unordered LiveLesson map.
	// Unfortunately this map is pretty well embedded in the API at this point, and would be a lot of work on antidote-core and antidote-web
	// to convert this to an ordered data structure like a slice.
	//
	// In most cases, this order doesn't really matter. This code however, is a huge exception - the order in which endpoint pods are created
	// is extremely important, as this is what determines which pods get which IP addresses from the specified subnet.
	//
	// Because this is the only place I'm currently aware of that this order matters, I've inserted this simple logic to create a slice of strings that
	// represent the original order in which these endpoints appeared, and the subsequent loop can use this to create pods in the original order from
	// the lesson definition. HOWEVER, if we discover that other use cases exist that require consistent ordering, we should tackle the conversion
	// of this field to a slice, and remove this workaround.
	epOrdered := []string{}
	for _, lep := range lesson.Endpoints {
		epOrdered = append(epOrdered, lep.Name)
	}

	// Create pods and services
	for _, epName := range epOrdered {
		ep := ll.LiveEndpoints[epName]

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
	cp := &sync.Mutex{}

	failLesson := false

	for name, pod := range createdPods {
		go func(sc ot.SpanContext, name string, pod *corev1.Pod) {
			span := ot.StartSpan("scheduler_pod_status", ot.ChildOf(sc))
			defer span.Finish()
			span.SetTag("podName", name)
			defer wg.Done()

			for i := 0; i < 150; i++ {
				rdy, err := s.getPodStatus(pod)
				if err != nil {
					s.recordPodLogs(span.Context(), ll.ID, pod.Name, initContainerName)
					s.recordPodLogs(span.Context(), ll.ID, pod.Name, "")
					failLesson = true
					return
				}

				if rdy {
					cp.Lock()
					delete(createdPods, name)
					cp.Unlock()
					return
				}

				time.Sleep(2 * time.Second)
			}

			// We would only get to this point if the pod failed to start in the first place.
			// One potential reason for this is a failure in the init container, so we should attempt
			// to gather those logs.
			s.recordPodLogs(span.Context(), ll.ID, pod.Name, initContainerName)

			err = fmt.Errorf("Timed out waiting for %s to start", name)
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			failLesson = true
			return
		}(span.Context(), name, pod)
	}

	wg.Wait()

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
