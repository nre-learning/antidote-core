package api

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	influx "github.com/influxdata/influxdb/client/v2"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

func (s *SyringeAPIServer) recordProvisioningTime(res *scheduler.LessonScheduleResult) error {

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     s.SyringeConfig.InfluxURL,
		Username: s.SyringeConfig.InfluxUsername,
		Password: s.SyringeConfig.InfluxPassword,

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
		"lessonSlug":  res.Lesson.Slug,
		"lessonName":  res.Lesson.LessonName,
		"syringeTier": s.SyringeConfig.Tier,
		"syringeId":   s.SyringeConfig.SyringeID,
	}

	fields := map[string]interface{}{
		"lessonSlug":       res.Lesson.Slug,
		"provisioningTime": timeSecs,
		"lessonName":       res.Lesson.LessonName,
		"lessonSlugName":   fmt.Sprintf("%s - %s", res.Lesson.Slug, res.Lesson.LessonName),
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

func (s *SyringeAPIServer) startTSDBExport() error {

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: s.SyringeConfig.InfluxURL,

		Username: s.SyringeConfig.InfluxUsername,
		Password: s.SyringeConfig.InfluxPassword,

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
			tags["syringeTier"] = s.SyringeConfig.Tier
			tags["syringeId"] = s.SyringeConfig.SyringeID

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

	return nil
}

func (s *SyringeAPIServer) getCountAndDuration(lessonSlug string) (int64, int64) {

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

		tts, err := ptypes.Timestamp(liveLesson.CreatedTime)
		if err != nil {
			log.Errorf("Problem converting timestamp: %v", err)
			log.Error(liveLesson.CreatedTime)
		}
		durations = append(durations, int64(time.Since(tts)*time.Second))
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
