package scheduler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	pb "github.com/nre-learning/syringe/api/exp/generated"
)

type LessonScheduleRequest struct {
	Operation    OperationType
	LiveLessonID string
	Stage        int32
	Created      time.Time
}

type LessonScheduleResult struct {
	Success          bool
	Stage            int32
	Lesson           *pb.Lesson
	Operation        OperationType
	Message          string
	ProvisioningTime int
	Uuid             string
}

func (ls *LessonScheduler) handleRequestCREATE(newRequest *LessonScheduleRequest) {

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	// TODO(mierdin): This function returns a pointer, and as a result, we're able to
	// set properties of newKubeLab and the map that stores these is immediately updated.
	// This isn't technically goroutine-safe, even though it's more or lesson guaranteed
	// that only one goroutine will access THIS pointer (since they're separated by
	// session ID). Not currently encountering issues here, but might want to think about it
	// longer-term.
	err := ls.createK8sStuff(newRequest)
	if err != nil {
		log.Errorf("Error creating lesson: %s", err)
		ls.Results <- &LessonScheduleResult{
			Success:   false,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
		}
		return
	}

	// TODO the request above should only contain a livelesson ID for us to look up. We should first GET that
	// from the datastore and proceed only if successful.

	// INITIAL_BOOT is the default status, but sending this to the API after Kubelab creation will
	// populate the entry in the API server with data about endpoints that it doesn't have yet.
	log.Debugf("Kubelab creation for %s complete. Moving into INITIAL_BOOT status", newRequest.Uuid)
	newKubeLab.Status = pb.Status_INITIAL_BOOT
	newKubeLab.CurrentStage = newRequest.Stage
	ls.setKubelab(newRequest.Uuid, newKubeLab)

	// Trigger a status update in the API server
	ls.Results <- &LessonScheduleResult{
		Success:   true,
		Lesson:    newRequest.Lesson,
		Uuid:      newRequest.Uuid,
		Operation: OperationType_MODIFY,
		Stage:     newRequest.Stage,
	}

	liveLesson := newKubeLab.ToLiveLesson()

	var success = false
	for i := 0; i < 600; i++ {
		time.Sleep(1 * time.Second)

		log.Debugf("About to test endpoint reachability for livelesson %s with endpoints %v", liveLesson.LessonUUID, liveLesson.LiveEndpoints)

		reachability := ls.testEndpointReachability(liveLesson)

		log.Debugf("Livelesson %s health check results: %v", liveLesson.LessonUUID, reachability)

		// Update reachability status
		failed := false
		healthy := 0
		total := len(reachability)
		for _, reachable := range reachability {
			if reachable {
				healthy++
			} else {
				failed = true
			}
		}
		newKubeLab.HealthyTests = healthy
		newKubeLab.TotalTests = total

		// Trigger a status update in the API server
		ls.Results <- &LessonScheduleResult{
			Success:   true,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: OperationType_MODIFY,
			Stage:     newRequest.Stage,
		}

		// Begin again if one of the endpoints isn't reachable
		if failed {
			continue
		}

		success = true
		break

	}

	if !success {
		log.Errorf("Timeout waiting for lesson %d to become reachable", newRequest.Lesson.LessonId)
		ls.Results <- &LessonScheduleResult{
			Success:   false,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
			Stage:     newRequest.Stage,
		}
		return
	} else {

		log.Debugf("Setting status for livelesson %s to CONFIGURATION", newRequest.Uuid)
		newKubeLab.Status = pb.Status_CONFIGURATION

		// Trigger a status update in the API server
		ls.Results <- &LessonScheduleResult{
			Success:   true,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: OperationType_MODIFY,
			Stage:     newRequest.Stage,
		}
	}

	log.Infof("Performing configuration for new instance of lesson %d", newRequest.Lesson.LessonId)
	err = ls.configureStuff(nsName, liveLesson, newRequest)
	if err != nil {
		ls.Results <- &LessonScheduleResult{
			Success:   false,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
			Stage:     newRequest.Stage,
		}
	} else {
		log.Debugf("Setting %s to READY", newRequest.Uuid)
		newKubeLab.Status = pb.Status_READY

		ls.Results <- &LessonScheduleResult{
			Success:          true,
			Lesson:           newRequest.Lesson,
			Uuid:             newRequest.Uuid,
			ProvisioningTime: int(time.Since(newRequest.Created).Seconds()),
			Operation:        newRequest.Operation,
			Stage:            newRequest.Stage,
		}
	}

	// Set network policy ONLY after configuration has had a chance to take place. Once this is in place,
	// only config pods spawned by Jobs will have internet access, so if this takes place earlier, lessons
	// won't initially come up at all.
	if !ls.SyringeConfig.AllowEgress {
		ls.createNetworkPolicy(nsName)
	}

	ls.setKubelab(newRequest.Uuid, newKubeLab)

}

func (ls *LessonScheduler) handleRequestMODIFY(newRequest *LessonScheduleRequest) {

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	// TODO(mierdin): Check for presence first?
	kl := ls.KubeLabs[newRequest.Uuid]
	kl.CurrentStage = newRequest.Stage
	ls.setKubelab(newRequest.Uuid, kl)

	liveLesson := kl.ToLiveLesson()

	log.Infof("Performing configuration of modified instance of lesson %d", newRequest.Lesson.LessonId)
	err := ls.configureStuff(nsName, liveLesson, newRequest)
	if err != nil {
		ls.Results <- &LessonScheduleResult{
			Success:   false,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
			Stage:     newRequest.Stage,
		}
		return
	}

	err = ls.boopNamespace(nsName)
	if err != nil {
		log.Errorf("Problem modify-booping %s: %v", nsName, err)
	}

	ls.Results <- &LessonScheduleResult{
		Success:   true,
		Lesson:    newRequest.Lesson,
		Uuid:      newRequest.Uuid,
		Operation: newRequest.Operation,
		Stage:     newRequest.Stage,
	}
}

func (ls *LessonScheduler) handleRequestBOOP(newRequest *LessonScheduleRequest) {
	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	err := ls.boopNamespace(nsName)
	if err != nil {
		log.Errorf("Problem booping %s: %v", nsName, err)
	}
}

func (ls *LessonScheduler) handleRequestDELETE(newRequest *LessonScheduleRequest) {

	// Delete the namespace object and then clean up our local state
	// TODO(mierdin): This is an unlikely operation to fail, but maybe add some kind logic here just in case?
	ls.deleteNamespace(fmt.Sprintf("%s-ns", newRequest.Uuid))
	ls.deleteKubelab(newRequest.Uuid)

	ls.Results <- &LessonScheduleResult{
		Success:   true,
		Uuid:      newRequest.Uuid,
		Operation: newRequest.Operation,
	}
}

func (ls *LessonScheduler) createK8sStuff(req *LessonScheduleRequest) error {

	ns, err := ls.createNamespace(req)
	if err != nil {
		log.Error(err)
	}

	err = ls.syncSecret(ns.ObjectMeta.Name)
	if err != nil {
		log.Error("Unable to sync secret into this namespace. Ingress-based resources (like iframes) may not work.")
	}

	// Append endpoint and create ingress for jupyter lab guide if necessary
	if usesJupyterLabGuide(req.Lesson) {

		jupyterEp := &pb.Endpoint{
			Name:            "jupyterlabguide",
			Image:           fmt.Sprintf("antidotelabs/jupyter:%s", ls.BuildInfo["imageVersion"]),
			AdditionalPorts: []int32{8888},
		}
		req.Lesson.Endpoints = append(req.Lesson.Endpoints, jupyterEp)

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
