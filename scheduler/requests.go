package scheduler

import (
	"fmt"
	"time"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	log "github.com/sirupsen/logrus"
)

type LessonScheduleRequest struct {
	Lesson    *pb.Lesson
	Operation OperationType
	Uuid      string
	Stage     int32
	Created   time.Time
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
	newKubeLab, err := ls.createKubeLab(newRequest)
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

		epr := ls.testEndpointReachability(liveLesson)

		log.Debugf("Livelesson %s health check results: %v", liveLesson.LessonUUID, epr)

		// Update reachability status
		endpointUnreachable := false
		for epName, reachable := range epr {
			if reachable {
				newKubeLab.setEndpointReachable(epName)
			} else {
				endpointUnreachable = true
			}
		}

		// Trigger a status update in the API server
		ls.Results <- &LessonScheduleResult{
			Success:   true,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: OperationType_MODIFY,
			Stage:     newRequest.Stage,
		}

		// Begin again if one of the endpoints isn't reachable
		if endpointUnreachable {
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

	if HasDevices(newRequest.Lesson) {
		log.Infof("Performing configuration for new instance of lesson %d", newRequest.Lesson.LessonId)
		err := ls.configureStuff(nsName, liveLesson, newRequest)
		if err != nil {
			ls.Results <- &LessonScheduleResult{
				Success:   false,
				Lesson:    newRequest.Lesson,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
				Stage:     newRequest.Stage,
			}
		}
	} else {
		log.Infof("Nothing to configure in %s", newRequest.Uuid)
	}

	// Set network policy ONLY after configuration has had a chance to take place. Once this is in place,
	// only config pods spawned by Jobs will have internet access, so if this takes place earlier, lessons
	// won't initially come up at all.
	ls.createNetworkPolicy(nsName)

	ls.setKubelab(newRequest.Uuid, newKubeLab)

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

func (ls *LessonScheduler) handleRequestMODIFY(newRequest *LessonScheduleRequest) {

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	// TODO(mierdin): Check for presence first?
	kl := ls.KubeLabs[newRequest.Uuid]
	kl.CurrentStage = newRequest.Stage
	ls.setKubelab(newRequest.Uuid, kl)

	liveLesson := kl.ToLiveLesson()

	if HasDevices(newRequest.Lesson) {
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
	} else {
		log.Infof("Skipping configuration of modified instance of lesson %d", newRequest.Lesson.LessonId)
	}

	err := ls.boopNamespace(nsName)
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

func (ls *LessonScheduler) handleRequestVERIFY(newRequest *LessonScheduleRequest) {
	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	ls.killAllJobs(nsName, "verify")
	verifyJob, err := ls.verifyLiveLesson(newRequest)
	if err != nil {
		log.Debugf("Unable to verify: %s", err)

		ls.Results <- &LessonScheduleResult{
			Success:   false,
			Lesson:    newRequest.Lesson,
			Uuid:      newRequest.Uuid,
			Operation: newRequest.Operation,
			Stage:     newRequest.Stage,
		}
	}

	// Quick timeout here. About 30 seconds or so.
	for i := 0; i < 15; i++ {

		finished, err := ls.verifyStatus(verifyJob, newRequest)
		// Return immediately if there was a problem
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

		// Return immediately if successful and finished
		if finished == true {
			ls.Results <- &LessonScheduleResult{
				Success:   true,
				Lesson:    newRequest.Lesson,
				Uuid:      newRequest.Uuid,
				Operation: newRequest.Operation,
				Stage:     newRequest.Stage,
			}
			return
		}

		// Not failed or succeeded yet. Try again.
		time.Sleep(2 * time.Second)
	}

	// Return failure, there's clearly a problem.
	ls.Results <- &LessonScheduleResult{
		Success:   false,
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
