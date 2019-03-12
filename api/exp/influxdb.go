package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	influx "github.com/influxdata/influxdb/client/v2"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

var (
	influxURL    = "http://influxdb:8086"
	influxUDPUrl = "influxdb:8089"
)

func (s *server) recordProvisioningTime(timeSecs int, res *scheduler.LessonScheduleResult) error {

	// Make client
	c, err := influx.NewUDPClient(influx.UDPConfig{
		Addr: influxUDPUrl,
	})
	if err != nil {
		log.Error("Error creating InfluxDB UDP Client: ", err.Error())
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
		"lessonId":   strconv.Itoa(int(res.LessonDef.LessonId)),
		"lessonName": res.LessonDef.LessonName,
	}

	fields := map[string]interface{}{
		"lessonId":         strconv.Itoa(int(res.LessonDef.LessonId)),
		"provisioningTime": timeSecs,
		"lessonName":       res.LessonDef.LessonName,
		"lessonIDName":     fmt.Sprintf("%d - %s", res.LessonDef.LessonId, res.LessonDef.LessonName),
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

func (s *server) startTSDBExport() error {

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: influxURL,
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

		for lessonId, _ := range s.scheduler.LessonDefs {

			tags := map[string]string{}
			fields := map[string]interface{}{}

			tags["lessonId"] = strconv.Itoa(int(lessonId))
			tags["lessonName"] = s.scheduler.LessonDefs[lessonId].LessonName

			count, duration := s.getCountAndDuration(lessonId)
			fields["lessonName"] = s.scheduler.LessonDefs[lessonId].LessonName
			fields["lessonId"] = strconv.Itoa(int(lessonId))

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

func (s *server) getCountAndDuration(lessonId int32) (int64, int64) {

	count := 0

	durations := []int64{}
	for _, liveLesson := range s.liveLessonState {
		if liveLesson.LessonId != lessonId {
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
