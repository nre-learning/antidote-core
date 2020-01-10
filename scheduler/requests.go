package scheduler

import (
	"fmt"
	"time"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	opentracing "github.com/opentracing/opentracing-go"
)

type LessonScheduleRequest struct {
	Lesson    *pb.Lesson
	Operation OperationType
	Session   string // This was added to keep tracing similar between API server and Scheduler
	Uuid      string
	Stage     int32
	Created   time.Time
	APISpan   opentracing.Span
}

type LessonScheduleResult struct {
	Success          bool
	Stage            int32
	Lesson           *pb.Lesson
	Operation        OperationType
	Message          string
	ProvisioningTime int
	Uuid             string
	SchedulerSpan    opentracing.Span
}

func (ls *LessonScheduler) handleRequestCREATE(newRequest *LessonScheduleRequest) {

	span := opentracing.StartSpan(
		"livelesson_scheduler_request_create",
		opentracing.ChildOf(newRequest.APISpan.Context()))
	defer span.Finish()

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	// TODO(mierdin): This function returns a pointer, and as a result, we're able to
	// set properties of newKubeLab and the map that stores these is immediately updated.
	// This isn't technically goroutine-safe, even though it's more or less guaranteed
	// that only one goroutine will access THIS pointer (since they're separated by
	// session ID). Not currently encountering issues here, but might want to think about it
	// longer-term.
	newKubeLab, err := ls.createKubeLab(newRequest)
	if err != nil {
		span.LogEvent(fmt.Sprintf("Error creating kubelab: %s", err))
		span.SetTag("error", true)
		ls.Results <- &LessonScheduleResult{
			Success:       false,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     newRequest.Operation,
			SchedulerSpan: span,
		}
		return
	}

	span.LogEvent("Kubelab creation complete. Moving into INITIAL_BOOT status")

	// INITIAL_BOOT is the default status, but sending this to the API after Kubelab creation will
	// populate the entry in the API server with data about endpoints that it doesn't have yet.
	newKubeLab.Status = pb.Status_INITIAL_BOOT
	newKubeLab.CurrentStage = newRequest.Stage
	ls.setKubelab(newRequest.Uuid, newKubeLab)

	// Trigger a status update in the API server
	ls.Results <- &LessonScheduleResult{
		Success:       true,
		Lesson:        newRequest.Lesson,
		Uuid:          newRequest.Uuid,
		Operation:     OperationType_MODIFY,
		Stage:         newRequest.Stage,
		SchedulerSpan: span,
	}

	liveLesson := newKubeLab.ToLiveLesson()

	var success = false
	for i := 0; i < 600; i++ {
		time.Sleep(1 * time.Second)

		reachability := ls.testEndpointReachability(liveLesson)
		span.LogEventWithPayload("Health check results", reachability)

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
			Success:       true,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     OperationType_MODIFY,
			Stage:         newRequest.Stage,
			SchedulerSpan: span,
		}

		// Begin again if one of the endpoints isn't reachable
		if failed {
			continue
		}

		success = true
		break

	}

	if !success {
		span.LogEvent("Timeout waiting for LiveLesson to become reachable")
		span.SetTag("error", true)
		ls.Results <- &LessonScheduleResult{
			Success:       false,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     newRequest.Operation,
			Stage:         newRequest.Stage,
			SchedulerSpan: span,
		}
		return
	} else {

		span.LogEvent("Setting status to CONFIGURATION")
		newKubeLab.Status = pb.Status_CONFIGURATION

		// Trigger a status update in the API server
		ls.Results <- &LessonScheduleResult{
			Success:       true,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     OperationType_MODIFY,
			Stage:         newRequest.Stage,
			SchedulerSpan: span,
		}
	}

	span.LogEvent("Configuring livelesson endpoints")
	err = ls.configureStuff(nsName, liveLesson, newRequest)
	if err != nil {
		span.LogEvent("Encountered an unrecoverable error configuring lesson")
		span.SetTag("error", true)
		ls.Results <- &LessonScheduleResult{
			Success:       false,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     newRequest.Operation,
			Stage:         newRequest.Stage,
			SchedulerSpan: span,
		}
	} else {
		span.LogEvent("Setting status to READY")
		newKubeLab.Status = pb.Status_READY

		ls.Results <- &LessonScheduleResult{
			Success:          true,
			Lesson:           newRequest.Lesson,
			Uuid:             newRequest.Uuid,
			ProvisioningTime: int(time.Since(newRequest.Created).Seconds()),
			Operation:        newRequest.Operation,
			Stage:            newRequest.Stage,
			SchedulerSpan:    span,
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
	span := opentracing.StartSpan(
		"livelesson_scheduler_request_modify",
		opentracing.ChildOf(newRequest.APISpan.Context()))
	defer span.Finish()

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	// TODO(mierdin): Check for presence first?
	kl := ls.KubeLabs[newRequest.Uuid]
	kl.CurrentStage = newRequest.Stage
	ls.setKubelab(newRequest.Uuid, kl)

	liveLesson := kl.ToLiveLesson()

	span.LogEvent("Configuring livelesson endpoints")
	err := ls.configureStuff(nsName, liveLesson, newRequest)
	if err != nil {
		span.LogEvent("Encountered an unrecoverable error configuring lesson")
		span.SetTag("error", true)
		ls.Results <- &LessonScheduleResult{
			Success:       false,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     newRequest.Operation,
			Stage:         newRequest.Stage,
			SchedulerSpan: span,
		}
		return
	}

	err = ls.boopNamespace(nsName)
	if err != nil {
		span.LogEvent(fmt.Sprintf("Problem modify-booping %s: %v", nsName, err))
		// Not currently doing anything to handle this, just noting it
	}

	ls.Results <- &LessonScheduleResult{
		Success:       true,
		Lesson:        newRequest.Lesson,
		Uuid:          newRequest.Uuid,
		Operation:     newRequest.Operation,
		Stage:         newRequest.Stage,
		SchedulerSpan: span,
	}

}

func (ls *LessonScheduler) handleRequestVERIFY(newRequest *LessonScheduleRequest) {
	span := opentracing.StartSpan(
		"livelesson_scheduler_request_verify",
		opentracing.ChildOf(newRequest.APISpan.Context()))
	defer span.Finish()

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	ls.killAllJobs(nsName, "verify")
	verifyJob, err := ls.verifyLiveLesson(newRequest)
	if err != nil {
		ls.Results <- &LessonScheduleResult{
			Success:       false,
			Lesson:        newRequest.Lesson,
			Uuid:          newRequest.Uuid,
			Operation:     newRequest.Operation,
			Stage:         newRequest.Stage,
			SchedulerSpan: span,
		}
	}

	// Quick timeout here. About 30 seconds or so.
	for i := 0; i < 15; i++ {

		finished, err := ls.verifyStatus(verifyJob, newRequest)
		// Return immediately if there was a problem
		if err != nil {
			ls.Results <- &LessonScheduleResult{
				Success:       false,
				Lesson:        newRequest.Lesson,
				Uuid:          newRequest.Uuid,
				Operation:     newRequest.Operation,
				Stage:         newRequest.Stage,
				SchedulerSpan: span,
			}
			return
		}

		// Return immediately if successful and finished
		if finished == true {
			ls.Results <- &LessonScheduleResult{
				Success:       true,
				Lesson:        newRequest.Lesson,
				Uuid:          newRequest.Uuid,
				Operation:     newRequest.Operation,
				Stage:         newRequest.Stage,
				SchedulerSpan: span,
			}
			return
		}

		// Not failed or succeeded yet. Try again.
		time.Sleep(2 * time.Second)
	}

	// Return failure, there's clearly a problem.
	ls.Results <- &LessonScheduleResult{
		Success:       false,
		Lesson:        newRequest.Lesson,
		Uuid:          newRequest.Uuid,
		Operation:     newRequest.Operation,
		Stage:         newRequest.Stage,
		SchedulerSpan: span,
	}
}

func (ls *LessonScheduler) handleRequestBOOP(newRequest *LessonScheduleRequest) {
	span := opentracing.StartSpan(
		"livelesson_scheduler_request_boop",
		opentracing.ChildOf(newRequest.APISpan.Context()))
	defer span.Finish()

	nsName := fmt.Sprintf("%s-ns", newRequest.Uuid)

	err := ls.boopNamespace(nsName)
	if err != nil {
		span.LogEvent(fmt.Sprintf("Problem booping %s: %v", nsName, err))
	}
}

func (ls *LessonScheduler) handleRequestDELETE(newRequest *LessonScheduleRequest) {
	span := opentracing.StartSpan(
		"livelesson_scheduler_request_delete",
		opentracing.ChildOf(newRequest.APISpan.Context()))
	defer span.Finish()

	// Delete the namespace object and then clean up our local state
	// TODO(mierdin): This is an unlikely operation to fail, but maybe add some kind logic here just in case?
	ls.deleteNamespace(fmt.Sprintf("%s-ns", newRequest.Uuid))
	ls.deleteKubelab(newRequest.Uuid)

	ls.Results <- &LessonScheduleResult{
		Success:       true,
		Uuid:          newRequest.Uuid,
		Operation:     newRequest.Operation,
		SchedulerSpan: span,
	}

}
