package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	nats "github.com/nats-io/nats.go"
	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	"github.com/nre-learning/antidote-core/services"
	"github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
)

// AntidoteStats tracks lesson startup time, as well as periodically exports usage data to a TSDB
type AntidoteStats struct {
	NC     *nats.Conn
	NEC    *nats.EncodedConn
	Config config.AntidoteConfig
	Db     db.DataManager
}

// Start starts the AntidoteStats service
func (s *AntidoteStats) Start() error {

	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan("stats_root")
	defer span.Finish()

	// Begin periodically exporting metrics to TSDB
	go s.startTSDBExport(span.Context())

	s.NC.Subscribe("antidote.lsr.completed", func(msg *nats.Msg) {
		t := services.NewTraceMsg(msg)
		tracer := opentracing.GlobalTracer()
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			log.Printf("Extract error: %v", err)
		}

		span := tracer.StartSpan(
			"scheduler_lsr_incoming",
			opentracing.ChildOf(sc))
		defer span.Finish()

		rem := t.Bytes()
		var lsr services.LessonScheduleRequest
		_ = json.Unmarshal(rem, &lsr)
		s.recordProvisioningTime(span.Context(), lsr)

	})

	// Wait forever
	ch := make(chan struct{})
	<-ch

	return nil
}

func (s *AntidoteStats) recordProvisioningTime(sc opentracing.SpanContext, res services.LessonScheduleRequest) error {

	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"stats_record_provisioning",
		opentracing.ChildOf(sc))
	defer span.Finish()

	lesson, err := s.Db.GetLesson(span.Context(), res.LessonSlug)
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

	q := influx.NewQuery("CREATE DATABASE antidote_metrics", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		//
	}

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  "antidote_metrics",
		Precision: "s",
	})
	if err != nil {
		log.Error("Unable to create metrics batch point: ", err)
		return err
	}

	// Create a point and add to batch
	tags := map[string]string{
		"lessonSlug":   res.LessonSlug,
		"lessonName":   lesson.Name,
		"antidoteTier": s.Config.Tier,
		"antidoteId":   s.Config.InstanceID,
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

func (s *AntidoteStats) startTSDBExport(sc opentracing.SpanContext) error {

	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"stats_periodic_export",
		opentracing.ChildOf(sc))
	defer span.Finish()

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

	q := influx.NewQuery("CREATE DATABASE antidote_metrics", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		//
	}

	for {
		time.Sleep(1 * time.Minute)

		log.Debug("Recording periodic influxdb metrics")

		// Create a new point batch
		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:  "antidote_metrics",
			Precision: "s",
		})
		if err != nil {
			log.Error("Unable to create metrics batch point: ", err)
			continue
		}

		lessons, err := s.Db.ListLessons(span.Context())
		if err != nil {
			log.Error("Unable to get lessons from DB for influxdb")
		}

		for _, lesson := range lessons {

			tags := map[string]string{}
			fields := map[string]interface{}{}

			tags["lessonSlug"] = lesson.Slug
			tags["lessonName"] = lesson.Name
			tags["antidoteTier"] = s.Config.Tier
			tags["antidoteId"] = s.Config.InstanceID

			count, duration := s.getCountAndDuration(span.Context(), lesson.Slug)
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

func (s *AntidoteStats) getCountAndDuration(sc opentracing.SpanContext, lessonSlug string) (int64, int64) {
	// Don't bother opening a new span for this function, just pass to the underlying DB call

	count := 0

	liveLessons, err := s.Db.ListLiveLessons(sc)
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
