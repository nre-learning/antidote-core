package api

import (
	scheduler "github.com/nre-learning/syringe/scheduler"
)

// Because we're still running the influx code in the API package, this function is the ONLY reason we even
// have a response infrastructure still in place. Once influx is moved into its own service, and this
// function is removed, please also remove all of the response infrastructure, as the API server will at
// that point no longer need it at all.
func (s *SyringeAPIServer) handleResultCREATE(result *scheduler.LessonScheduleResult) {
	if s.SyringeConfig.InfluxdbEnabled {
		s.recordProvisioningTime(result)
	}
}
