package scheduler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	models "github.com/nre-learning/syringe/db/models"
)

type LessonScheduleRequest struct {
	Operation     OperationType
	LessonSlug    string
	LiveLessonID  string
	LiveSessionID string
	Stage         int32
	Created       time.Time
}

// TODO(mierdin) This needs to be removed once the influx stuff is re-thought
// currently, that's the only thing that is using this.
type LessonScheduleResult struct {
	LessonSlug       string
	LiveLessonID     string
	LiveSessionID    string
	ProvisioningTime int
}

func (ls *LessonScheduler) handleRequestCREATE(newRequest *LessonScheduleRequest) {

	nsName := generateNamespaceName(ls.SyringeConfig.SyringeID, newRequest.LiveLessonID)

	ll, err := ls.Db.GetLiveLesson(newRequest.LiveLessonID)
	if err != nil {
		log.Errorf("Error getting livelesson: %v", err)
		return
	}

	err = ls.createK8sStuff(newRequest)
	if err != nil {
		log.Errorf("Error creating lesson: %v", err)
		return
	}

	log.Debugf("Bootstrap complete for livelesson %s. Moving into BOOTING status", newRequest.LiveLessonID)
	ll.Status = models.Status_BOOTING
	ll.LessonStage = newRequest.Stage
	err = ls.Db.UpdateLiveLesson(ll)
	if err != nil {
		log.Errorf("Error updating livelesson: %v", err)
		return
	}

	var success = false
	for i := 0; i < 600; i++ {
		time.Sleep(1 * time.Second)

		log.Debugf("About to test endpoint reachability for livelesson %s with endpoints %v", ll.ID, ll.LiveEndpoints)

		reachability := ls.testEndpointReachability(ll)

		log.Debugf("Livelesson %s health check results: %v", ll.ID, reachability)

		// Update reachability status
		failed := false
		healthy := int32(0)
		total := int32(len(reachability))
		for _, reachable := range reachability {
			if reachable {
				healthy++
			} else {
				failed = true
			}
		}
		ll.HealthyTests = healthy
		ll.TotalTests = total

		// Begin again if one of the endpoints isn't reachable
		if failed {
			continue
		}

		success = true
		break

	}

	if !success {
		log.Errorf("Timeout waiting for livelesson to become reachable", ll.ID)
		ll.Error = true
		err = ls.Db.UpdateLiveLesson(ll)
		if err != nil {
			log.Errorf("Error updating livelesson: %v", err)
		}
		return
	}

	log.Debugf("Setting status for livelesson %s to CONFIGURATION", newRequest.LiveLessonID)
	ll.Status = models.Status_CONFIGURATION
	err = ls.Db.UpdateLiveLesson(ll)
	if err != nil {
		log.Errorf("Error updating livelesson %s: %v", ll.ID, err)
	}

	log.Infof("Performing configuration for livelesson %s", ll.ID)
	err = ls.configureStuff(nsName, ll, newRequest)
	if err != nil {
		log.Errorf("Error configuring livelesson %s: %v", ll.ID, err)
		ll.Error = true
		err = ls.Db.UpdateLiveLesson(ll)
		if err != nil {
			log.Errorf("Error updating livelesson %s: %v", ll.ID, err)
		}
		return
	}

	// Set network policy ONLY after configuration has had a chance to take place. Once this is in place,
	// only config pods spawned by Jobs will have internet access, so if this takes place earlier, lessons
	// won't initially come up at all.
	if !ls.SyringeConfig.AllowEgress {
		ls.createNetworkPolicy(nsName)
	}

	log.Debugf("Setting livelesson %s to READY", newRequest.LiveLessonID)
	ll.Status = models.Status_READY
	err = ls.Db.UpdateLiveLesson(ll)
	if err != nil {
		log.Errorf("Error updating livelesson %s: %v", ll.ID, err)
	}
}

func (ls *LessonScheduler) handleRequestMODIFY(newRequest *LessonScheduleRequest) {

	nsName := generateNamespaceName(ls.SyringeConfig.SyringeID, newRequest.LiveLessonID)

	ll, err := ls.Db.GetLiveLesson(newRequest.LiveLessonID)
	if err != nil {
		log.Errorf("Error getting livelesson: %v", err)
		return
	}

	log.Infof("Performing configuration for stage %d of livelesson %s", newRequest.Stage, newRequest.LiveLessonID)
	err = ls.configureStuff(nsName, ll, newRequest)
	if err != nil {
		log.Errorf("Error configuring livelesson %s: %v", ll.ID, err)
		ll.Error = true
		err = ls.Db.UpdateLiveLesson(ll)
		if err != nil {
			log.Errorf("Error updating livelesson %s: %v", ll.ID, err)
		}
		return
	}

	err = ls.boopNamespace(nsName)
	if err != nil {
		log.Errorf("Problem modify-booping %s: %v", nsName, err)
	}
}

func (ls *LessonScheduler) handleRequestBOOP(newRequest *LessonScheduleRequest) {
	nsName := generateNamespaceName(ls.SyringeConfig.SyringeID, newRequest.LiveLessonID)

	err := ls.boopNamespace(nsName)
	if err != nil {
		log.Errorf("Problem booping %s: %v", nsName, err)
	}
}

// handleRequestDELETE handles a livelesson deletion request by first sending a delete request
// for the corresponding namespace, and then cleaning up local state.
func (ls *LessonScheduler) handleRequestDELETE(newRequest *LessonScheduleRequest) {
	nsName := generateNamespaceName(ls.SyringeConfig.SyringeID, newRequest.LiveLessonID)
	err := ls.deleteNamespace(nsName)
	if err != nil {
		log.Errorf("Unable to delete namespace %s: %v", nsName, err)
		return
	}
	err = ls.Db.DeleteLiveLesson(newRequest.LiveLessonID)
	if err != nil {
		log.Errorf("Error getting livelesson: %v", err)
	}
}

// createK8sStuff is a high-level workflow for creating all of the things necessary for a new instance
// of a livelesson. Pods, services, networks, networkpolicies, ingresses, etc to support a new running
// lesson are all created as part of this workflow.
func (ls *LessonScheduler) createK8sStuff(req *LessonScheduleRequest) error {

	ns, err := ls.createNamespace(req)
	if err != nil {
		log.Error(err)
	}

	err = ls.syncSecret(ns.ObjectMeta.Name)
	if err != nil {
		log.Error("Unable to sync secret into this namespace. Ingress-based resources (like iframes) may not work.")
	}

	lesson, err := ls.Db.GetLesson(req.LessonSlug)
	if err != nil {
		return err
	}

	ll, err := ls.Db.GetLiveLesson(req.LiveLessonID)
	if err != nil {
		return err
	}

	// Append endpoint and create ingress for jupyter lab guide if necessary
	if usesJupyterLabGuide(lesson) {

		jupyterEp := &models.LiveEndpoint{
			Name:  "jupyterlabguide",
			Image: fmt.Sprintf("antidotelabs/jupyter:%s", ls.BuildInfo["imageVersion"]),
			Ports: []int32{8888},
		}
		ll.LiveEndpoints = append(ll.LiveEndpoints, jupyterEp)

		_, err := ls.createIngress(
			ns.ObjectMeta.Name,
			jupyterEp,
			&pb.Presentation{
				Name: "web",
				Port: 8888,
			},
		)
		if err != nil {
			return fmt.Errorf("Unable to create ingress resource - %v", err)
		}
	}

	// TODO(mierdin): Should all of these details instead be dervived from LiveEndpoint? If so,
	// We'll need to make sure it's populated first at the API layer properly. We'll also have to
	// update the LiveEndpoint details like when the services get created

	// Create networks from connections property
	for c := range req.Lesson.Connections {
		connection := req.Lesson.Connections[c]
		_, err := ls.createNetwork(c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), req)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	// Create pods and services
	for d := range req.Lesson.Endpoints {
		ep := req.Lesson.Endpoints[d]

		// Create pod
		newPod, err := ls.createPod(
			ep,
			getMemberNetworks(ep.Name, req.Lesson.Connections),
			req,
		)
		if err != nil {
			log.Error(err)
			return err
		}

		// Expose via service if needed
		if len(newPod.Spec.Containers[0].Ports) > 0 {
			_, err := ls.createService(
				newPod,
				req,
			)
			if err != nil {
				log.Error(err)
				return err
			}
		}

		// Create appropriate presentations
		for pr := range ep.Presentations {
			p := ep.Presentations[pr]

			if p.Type == "http" {
				_, err := ls.createIngress(
					ns.ObjectMeta.Name,
					ep,
					p,
				)
				if err != nil {
					log.Error(err)
					return err
				}
			} else if p.Type == "ssh" {
				// nothing to do?
			}
		}
	}

	return nil
}
