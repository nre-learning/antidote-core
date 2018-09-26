package api

import (
	"fmt"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/ptypes"
	influx "github.com/influxdata/influxdb/client/v2"
	scheduler "github.com/nre-learning/syringe/scheduler"
)

var influxURL = "http://influxdb.default.svc.cluster.local:8086"

func (s *server) recordRequestTSDB(req *scheduler.LessonScheduleRequest) error {

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

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  "syringe_metrics",
		Precision: "s",
	})
	if err != nil {
		log.Error("Couldn't connect to Influxdb: ", err)
		return err
	}

	// Create a point and add to batch
	tags := map[string]string{
		"lessonId":   strconv.Itoa(int(req.LessonDef.LessonID)),
		"lessonName": req.LessonDef.LessonName,
		"sessionId":  req.Session,
		"operation":  string(req.Operation),
	}

	fields := map[string]interface{}{
		"lessonId":     strconv.Itoa(int(req.LessonDef.LessonID)),
		"sessionId":    req.Session,
		"operation":    req.Operation,
		"lessonName":   req.LessonDef.LessonName,
		"lessonIDName": fmt.Sprintf("%d - %s", req.LessonDef.LessonID, req.LessonDef.LessonName),
	}

	// can you get length of time used from namespace labels?

	// need to record current stage too

	pt, err := influx.NewPoint("lessonRequests", tags, fields, time.Now())
	if err != nil {
		log.Error("Error creating InfluxDB Point: ", err)
		return err
	}

	bp.AddPoint(pt)

	// Write the batch
	err = c.Write(bp)
	if err != nil {
		log.Error("Error writing InfluxDB Batch Points: ", err)
		return err
	}

	log.Debugf("Wrote request data to influxdb: %v", bp)

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

		// Create a new point batch
		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:  "syringe_metrics",
			Precision: "s",
		})
		if err != nil {
			log.Error("Couldn't connect to Influxdb: ", err)
			return err
		}

		/* Important to track:

		- How many livelessons are currently provisioned?
		- How many unique users are using the system?
		- How many
		*/

		// in-memory map of liveLessons, indexed by UUID
		// liveLessons map[string]*pb.LiveLesson

		// map of session IDs maps containing lesson ID and corresponding lesson UUID
		// sessions map[string]map[int32]string

		if len(s.liveLessons) == 0 {
			log.Debug("No active liveLessons, so skipping TSDB write")
			return nil
		}

		lessonIdMap := map[int32]string{}

		for _, liveLesson := range s.liveLessons {
			if _, ok := lessonIdMap[liveLesson.LessonId]; ok {
				continue // already present
			}

			lessonIdMap[liveLesson.LessonId] = "foo"
		}
		// apt-get update && apt-get install -y curl
		// curl -X POST 'http://10.104.168.244:8086/query?pretty=true' --data-urlencode "db=syringe_metrics" --data-urlencode "q=DROP DATABASE \"syringe_metrics\""
		// curl -G 'http://10.104.168.244:8086/query?pretty=true' --data-urlencode "db=syringe_metrics" --data-urlencode "q=select * from /sessionStatus/"

		for lessonId, _ := range lessonIdMap {

			tags := map[string]string{}
			fields := map[string]interface{}{}

			tags["lessonId"] = strconv.Itoa(int(lessonId))
			tags["lessonName"] = s.scheduler.LessonDefs[lessonId].LessonName

			count, duration := s.getCountAndDuration(lessonId)
			fields["lessonName"] = s.scheduler.LessonDefs[lessonId].LessonName
			fields["lessonId"] = strconv.Itoa(int(lessonId))
			fields["avgDuration"] = duration
			fields["activeNow"] = count

			pt, err := influx.NewPoint("sessionStatus", tags, fields, time.Now())
			if err != nil {
				log.Error("Error creating InfluxDB Point: ", err)
				return err
			}

			bp.AddPoint(pt)

		}

		// Write the batch
		err = c.Write(bp)
		if err != nil {
			log.Error("Error writing InfluxDB Batch Points: ", err)
			return err
		}

		log.Debugf("Wrote session data to influxdb: %v", bp)
	}

	return nil
}

func (s *server) getCountAndDuration(lessonId int32) (int64, int64) {

	count := 0

	durations := []int64{}
	for _, liveLesson := range s.liveLessons {
		if liveLesson.LessonId != lessonId {
			continue
		}

		count = count + 1

		tts, err := ptypes.Timestamp(liveLesson.CreatedTime)
		if err != nil {
			log.Errorf("Problem converting timestamp: %v", err)
		}
		durations = append(durations, int64(time.Since(tts)*time.Second))
	}

	total := int64(0)
	for i := range durations {
		total = total + durations[i]
	}

	return int64(count), total / int64(len(durations))
}
