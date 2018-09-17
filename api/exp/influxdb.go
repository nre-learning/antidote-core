package api

import (
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	influx "github.com/influxdata/influxdb/client/v2"
)

func (s *server) startTSDBExport() error {

	// Make client
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: "http://influxdb.default.svc.cluster.local:8086",
	})
	if err != nil {
		log.Error("Error creating InfluxDB Client: ", err.Error())
		return err
	}
	defer c.Close()

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

		for sessionId, lessonDetails := range s.sessions {

			for lessonId, _ := range lessonDetails {

				// Create a point and add to batch
				tags := map[string]string{
					"lessonId":  strconv.Itoa(int(lessonId)),
					"sessionId": sessionId,
				}

				fields := map[string]interface{}{
					"lessonId":  lessonId,
					"sessionId": sessionId,
				}

				// can you get length of time used from namespace labels?

				// need to record current stage too

				pt, err := influx.NewPoint("sessionStatus", tags, fields, time.Now())
				if err != nil {
					log.Error("Error creating InfluxDB Point: ", err)
					return err
				}

				bp.AddPoint(pt)
			}

		}

		// Write the batch
		err = c.Write(bp)
		if err != nil {
			log.Error("Error writing InfluxDB Batch Points: ", err)
			return err
		}

		log.Debug("Wrote session data to influxdb")
	}

	return nil
}
