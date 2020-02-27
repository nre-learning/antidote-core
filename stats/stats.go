package api

import (
	"errors"
	"fmt"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	nats "github.com/nats-io/nats.go"
	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	"github.com/nre-learning/antidote-core/services"
	log "github.com/sirupsen/logrus"
)

// AntidoteStats tracks lesson startup time, as well as periodically exports usage data to a TSDB
type AntidoteStats struct {
	NEC    *nats.EncodedConn
	Config config.AntidoteConfig
	Db     db.DataManager
}

// Start starts the AntidoteStats service
func (s *AntidoteStats) Start() error {

	// Begin periodically exporting metrics to TSDB
	go s.startTSDBExport()

	s.NEC.Subscribe("antidote.lsr.completed", func(lsr services.LessonScheduleRequest) {
		s.recordProvisioningTime(lsr)
	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}

func (s *AntidoteStats) recordProvisioningTime(res services.LessonScheduleRequest) error {

	lesson, err := s.Db.GetLesson(res.LessonSlug)
	if err != nil {
		return errors.New("Problem getting lesson details for recording provisioning time")
	}

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     s.Config.Stats.URL,
		Username: s.Config.Stats.Username,
		Password: s.Config.Stats.Password,

		// TODO(mierdin): Hopefully, temporary. Even though my influx instance is front-ended by a LetsEncrypt cert,
		// I was getting validation errors.
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Error("Error creating InfluxDB Client: ", err.Error())
		return err
	}
	defer c.Close()

	q := influx.NewQuery("CREATE DATABASE syringe_metrics", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		//
	}

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  "syringe_metrics",
		Precision: "s",
	})
	if err != nil {
		log.Error("Unable to create metrics batch point: ", err)
		return err
	}

	// Create a point and add to batch
	tags := map[string]string{
		"lessonSlug":  res.LessonSlug,
		"lessonName":  lesson.Name,
		"syringeTier": s.Config.Tier,
		"syringeId":   s.Config.InstanceID,
	}

	fields := map[string]interface{}{
		"lessonSlug":       res.LessonSlug,
		"provisioningTime": int(time.Since(res.Created).Seconds()),
		"lessonName":       lesson.Name,
		"lessonSlugName":   fmt.Sprintf("%s - %s", res.LessonSlug, lesson.Name),
	}

	pt, err := influx.NewPoint("provisioningTime", tags, fields, time.Now())
	if err != nil {
		log.Error("Error creating InfluxDB Point: ", err)
		return err
	}

	bp.AddPoint(pt)

	// Write the batch
	err = c.Write(bp)
	if err != nil {
		log.Warn("Unable to push provisioning time to Influx: ", err)
		return err
	}

	return nil
}

func (s *AntidoteStats) startTSDBExport() error {

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: s.Config.Stats.URL,

		Username: s.Config.Stats.Username,
		Password: s.Config.Stats.Password,

		// TODO(mierdin): Hopefully, temporary. Even though my influx instance is front-ended by a LetsEncrypt cert,
		// I was getting validation errors.
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Error("Error creating InfluxDB Client: ", err.Error())
		return err
	}
	defer c.Close()

	q := influx.NewQuery("CREATE DATABASE syringe_metrics", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		//
	}

	for {
		time.Sleep(1 * time.Minute)

		log.Debug("Recording periodic influxdb metrics")

		// Create a new point batch
		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:  "syringe_metrics",
			Precision: "s",
		})
		if err != nil {
			log.Error("Unable to create metrics batch point: ", err)
			continue
		}

		lessons, err := s.Db.ListLessons()
		if err != nil {
			log.Error("Unable to get lessons from DB for influxdb")
		}

		for _, lesson := range lessons {

			tags := map[string]string{}
			fields := map[string]interface{}{}

			tags["lessonSlug"] = lesson.Slug
			tags["lessonName"] = lesson.Name
			tags["syringeTier"] = s.Config.Tier
			tags["syringeId"] = s.Config.InstanceID

			count, duration := s.getCountAndDuration(lesson.Slug)
			fields["lessonName"] = lesson.Name
			fields["lessonSlug"] = lesson.Slug

			if duration != 0 {
				fields["avgDuration"] = duration
			}
			fields["activeNow"] = count

			// This is just for debugging, so only show active lessons
			if count > 0 {
				log.Debugf("Creating influxdb point: ID: %s | NAME: %s | ACTIVE: %d", fields["lessonId"], fields["lessonName"], count)
			}

			pt, err := influx.NewPoint("sessionStatus", tags, fields, time.Now())
			if err != nil {
				log.Error("Error creating InfluxDB Point: ", err)
				continue
			}

			bp.AddPoint(pt)

		}

		// Write the batch
		err = c.Write(bp)
		if err != nil {
			log.Warn("Unable to push periodic metrics to Influx: ", err)
			continue
		}
	}
}

func (s *AntidoteStats) getCountAndDuration(lessonSlug string) (int64, int64) {

	count := 0

	liveLessons, err := s.Db.ListLiveLessons()
	if err != nil {
		log.Errorf("Problem retrieving livelessons - %v", err)
		return 0, 0
	}

	durations := []int64{}
	for _, liveLesson := range liveLessons {
		if liveLesson.LessonSlug != lessonSlug {
			continue
		}

		count = count + 1
		durations = append(durations, int64(time.Since(liveLesson.CreatedTime)*time.Second))
	}

	total := int64(0)
	for i := range durations {
		total = total + durations[i]
	}

	if int64(len(durations)) == 0 {
		return int64(count), 0
	}

	return int64(count), total / int64(len(durations))
}
