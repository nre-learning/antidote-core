package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	influx "github.com/influxdata/influxdb/client/v2"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
	log "github.com/sirupsen/logrus"
)

type AntidoteStats struct {
	InfluxURL       string
	InfluxUsername  string
	InfluxPassword  string
	Curriculum      *pb.Curriculum
	LiveLessonState map[string]*pb.LiveLesson
	Tier            string
}

func (s *AntidoteStats) RecordProvisioningTime(timeSecs int, res *scheduler.LessonScheduleResult) error {

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     s.InfluxURL,
		Username: s.InfluxUsername,
		Password: s.InfluxPassword,

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
		"lessonId":    strconv.Itoa(int(res.Lesson.LessonId)),
		"lessonName":  res.Lesson.LessonName,
		"syringeTier": s.Tier,
	}

	fields := map[string]interface{}{
		"lessonId":         strconv.Itoa(int(res.Lesson.LessonId)),
		"provisioningTime": timeSecs,
		"lessonName":       res.Lesson.LessonName,
		"lessonIDName":     fmt.Sprintf("%d - %s", res.Lesson.LessonId, res.Lesson.LessonName),
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


func (s *AntidoteStats) StartTSDBExport() error {
	c, err := s.CreateInfluxClient()

	if err != nil {
		return err
	}
	defer c.Close()

	for {
		time.Sleep(1 * time.Minute)
		s.WriteBatchPoints(c)
	}

	return nil
}

func (s *AntidoteStats) CreateInfluxClient() (influx.Client, error) {
	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: s.InfluxURL,

		Username: s.InfluxUsername,
		Password: s.InfluxPassword,

		// TODO(mierdin): Hopefully, temporary. Even though my influx instance is front-ended by a LetsEncrypt cert,
		// I was getting validation errors.
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Error("Error creating InfluxDB Client: ", err.Error())
		return nil, err
	}

	q := influx.NewQuery("CREATE DATABASE syringe_metrics", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		//
	}

	return c, nil
}

func (s *AntidoteStats) WriteBatchPoints(c influx.Client) {
	log.Debug("Recording periodic influxdb metrics")

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  "syringe_metrics",
		Precision: "s",
	})
	if err != nil {
		log.Error("Unable to create metrics batch point: ", err)
		return
	}

	for _, liveLesson := range s.LiveLessonState {

		tags := map[string]string{}
		fields := map[string]interface{}{}

		tags["liveLessonUUID"] = liveLesson.LessonUUID
		tags["lessonId"] = strconv.Itoa(int(liveLesson.LessonId))
		tags["lessonName"] = s.Curriculum.Lessons[liveLesson.LessonId].LessonName
		tags["syringeTier"] = s.Tier

		fields["lessonName"] = s.Curriculum.Lessons[liveLesson.LessonId].LessonName
		fields["lessonId"] = strconv.Itoa(int(liveLesson.LessonId))
		fields["error"] = liveLesson.Error
		fields["healthyTests"] = liveLesson.HealthyTests
		fields["totalTests"] = liveLesson.TotalTests
		fields["lessonStage"] = liveLesson.LessonStage
		fields["createdTime"] = liveLesson.CreatedTime.Seconds

		pt, err := influx.NewPoint("sessionStatus", tags, fields, time.Now())
		if err != nil {
			log.Error("Error creating InfluxDB Point: ", err)
			return
		}

		bp.AddPoint(pt)

	}

	// Write the batch
	err = c.Write(bp)
	if err != nil {
		log.Warn("Unable to push periodic metrics to Influx: ", err)
		return
	}
}

func (s *AntidoteStats) getCountAndDuration(lessonId int32) (int64, int64) {
	count := 0
	durations := []int64{}

	for _, liveLesson := range s.LiveLessonState {
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

