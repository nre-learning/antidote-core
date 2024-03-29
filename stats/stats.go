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
	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/services"
	ot "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	logrus "github.com/sirupsen/logrus"
)

// AntidoteStats tracks lesson startup time, as well as periodically exports usage data to a TSDB
type AntidoteStats struct {
	NC     *nats.Conn
	Config config.AntidoteConfig
	Db     db.DataManager
}

// Start starts the AntidoteStats service
func (s *AntidoteStats) Start() error {

	// Begin periodically exporting metrics to TSDB
	go func() {
		err := s.startTSDBExport()
		if err != nil {
			logrus.Fatalf("Failed to start TSDB Export: %v", err)
		}
	}()

	s.NC.Subscribe(services.LsrCompleted, func(msg *nats.Msg) {
		t := services.NewTraceMsg(msg)
		tracer := ot.GlobalTracer()
		sc, err := tracer.Extract(ot.Binary, t)
		if err != nil {
			logrus.Errorf("Failed to extract for %s: %v", services.LsrCompleted, err)
		}

		span := tracer.StartSpan("stats_lsr_completed", ot.ChildOf(sc))
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

func (s *AntidoteStats) recordProvisioningTime(sc ot.SpanContext, res services.LessonScheduleRequest) error {
	span := ot.StartSpan("stats_record_provisioning", ot.ChildOf(sc))
	defer span.Finish()

	lesson, err := s.Db.GetLesson(span.Context(), res.LessonSlug)
	if err != nil {
		return errors.New("Problem getting lesson details for recording provisioning time")
	}

	ll, err := s.Db.GetLiveLesson(span.Context(), res.LiveLessonID)
	if err != nil {
		return errors.New("Problem getting lesson details for recording provisioning time")
	}

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:               s.Config.Stats.URL,
		Username:           s.Config.Stats.Username,
		Password:           s.Config.Stats.Password,
		InsecureSkipVerify: true,
	})
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
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
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
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
		"provisioningTime": int(time.Since(ll.CreatedTime).Seconds()),
		"lessonName":       lesson.Name,
		"lessonSlugName":   fmt.Sprintf("%s - %s", res.LessonSlug, lesson.Name),
	}

	pt, err := influx.NewPoint("provisioningTime", tags, fields, time.Now())
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	bp.AddPoint(pt)

	// Write the batch
	err = c.Write(bp)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	return nil
}

func (s *AntidoteStats) startTSDBExport() error {

	initSpan := ot.StartSpan("stats_periodic_export_start")
	defer initSpan.Finish()

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:               s.Config.Stats.URL,
		Username:           s.Config.Stats.Username,
		Password:           s.Config.Stats.Password,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	q := influx.NewQuery("CREATE DATABASE antidote_metrics", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		//
	}

	lessons, err := s.Db.ListLessons(initSpan.Context())
	if err != nil {
		initSpan.LogFields(log.Error(err))
		ext.Error.Set(initSpan, true)
		return err
	}

	initSpan.Finish()

	for {

		span := ot.StartSpan("stats_get_running_livelessons")
		time.Sleep(1 * time.Minute)

		// Create a new point batch
		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:  "antidote_metrics",
			Precision: "s",
		})
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			continue
		}

		liveLessons, err := s.Db.ListLiveLessons(span.Context())
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			continue
		}

		for _, lesson := range lessons {

			tags := map[string]string{}
			fields := map[string]interface{}{}

			tags["lessonSlug"] = lesson.Slug
			tags["lessonName"] = lesson.Name
			tags["antidoteTier"] = s.Config.Tier
			tags["antidoteId"] = s.Config.InstanceID

			count, duration := s.getCountAndDuration(liveLessons, lesson.Slug)
			fields["lessonName"] = lesson.Name
			fields["lessonSlug"] = lesson.Slug

			if duration != 0 {
				fields["avgDuration"] = duration
			}
			fields["activeNow"] = count

			span.LogFields(
				log.Object("fields", fields),
				log.Object("tags", tags),
				log.Int64("activeNow", count),
			)

			pt, err := influx.NewPoint("sessionStatus", tags, fields, time.Now())
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
				continue
			}

			bp.AddPoint(pt)

		}

		// Write the batch
		err = c.Write(bp)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			continue
		}

		span.Finish()
	}
}

func (s *AntidoteStats) getCountAndDuration(liveLessons map[string]models.LiveLesson, lessonSlug string) (int64, int64) {
	count := 0

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
