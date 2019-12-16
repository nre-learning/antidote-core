package main

import (
	"encoding/json"
	"fmt"
        "path/filepath"
        "reflect"
	"runtime"
	"testing"
	influx "github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"
	scheduler "github.com/nre-learning/syringe/scheduler"
)

func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

func initAntidoteStats() AntidoteStats {
	var mockSyringeConfig = GetmockSyringeConfig(true)
	var curriculum = GetCurriculum(mockSyringeConfig)
	var mockLiveLessonState = GetMockLiveLessonState()

	return AntidoteStats{
		InfluxURL:       mockSyringeConfig.InfluxURL,
		InfluxUsername:  mockSyringeConfig.InfluxUsername,
		InfluxPassword:  mockSyringeConfig.InfluxPassword,
		Curriculum:      curriculum,
		LiveLessonState: mockLiveLessonState,
	}
}

func createTestInfluxClient() (influx.Client, error) {
	mockSyringeConfig := GetmockSyringeConfig(true)
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:               mockSyringeConfig.InfluxURL,
		Username:           mockSyringeConfig.InfluxUsername,
		Password:           mockSyringeConfig.InfluxPassword,
		InsecureSkipVerify: true,
	})

	if err != nil {
		log.Error("Error creating InfluxDB Client: ", err.Error())
		return nil, err
	}

	return client, nil
}

func getData(client influx.Client, columns string, table string) ([][]interface{}, error) {
	query := influx.NewQuery(fmt.Sprintf("SELECT %s FROM %s", columns, table), "syringe_metrics", "")
	res, err := client.Query(query)

	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	if len(res.Results) > 0 {
		if len(res.Results[0].Series) > 0 {
			series := res.Results[0].Series[0]
			return series.Values, nil
		}
	}

	return make([][]interface{}, 0), nil
}

func dropTable(client influx.Client, table string) {
	query := influx.NewQuery(fmt.Sprintf("DELETE FROM %s", table), "syringe_metrics", "")
	_, err := client.Query(query)

	if err != nil {
		fmt.Println(err.Error())
	}
}

func TestStartTSDBExport(t *testing.T) {
	stats := initAntidoteStats()
	influxClient, err := stats.CreateInfluxClient()
	defer influxClient.Close()
	ok(t, err)

	dropTable(influxClient, "sessionStatus")
	stats.WriteBatchPoints(influxClient)
	data, err := getData(influxClient, "lessonId, liveLessonUUID", "sessionStatus")

	if err != nil {
		log.Error("Error querying data: ", err.Error())
		return
	}

	assert(t, len(data) > 0, "")
	assert(t, data[0][1] == "14", "")
	assert(t, data[0][2] == "14-4kfl6n3terlzxa3s", "")
}


func TestRecordProvisioningTime(t *testing.T) {
	var lessonId int32 = 14
	uuid := "14-4kfl6n3terlzxa3s"
	curriculum := GetCurriculum(GetmockSyringeConfig(true))
	lesson := curriculum.Lessons[lessonId]
	var provisioningTime int = 60
	res := &scheduler.LessonScheduleResult{
		Success: true,
		Stage:   1,
		Lesson:  lesson,
		Operation: scheduler.OperationType_CREATE,
		ProvisioningTime: provisioningTime,
		Uuid: uuid,
	}

	stats := initAntidoteStats()

	influxClient, testClientErr := createTestInfluxClient()
	if testClientErr != nil {
		log.Error("Error creating influxdb test client: ", testClientErr.Error())
		return
	}
	influxClient.Close();

	dropTable(influxClient, "provisioningTime")

	err := stats.RecordProvisioningTime(provisioningTime, res)
	ok(t, err)

	data, err := getData(influxClient, "lessonId, provisioningTime",  "provisioningTime")

        if err != nil {
                log.Error("Error querying data: ", err.Error())
                return
        }

	assert(t, len(data) > 0, "")
	assert(t, data[0][1] == "14", "")

	time, _ := json.Marshal(data[0][2])
	assert(t, string(time) == "60", "")
}
