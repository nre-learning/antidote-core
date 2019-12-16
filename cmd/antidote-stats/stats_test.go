package main

import (
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

func getRowCount(client influx.Client, table string) int {
	query := influx.NewQuery(fmt.Sprintf("SELECT * FROM %s", table), "syringe_metrics", "")
	res, err := client.Query(query)

	if err != nil {
		fmt.Println(err.Error())
	}

	if len(res.Results) > 0 {
		if len(res.Results[0].Series) > 0 {
			series := res.Results[0].Series[0]
			return len(series.Values)
		}
	}

	return 0
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
	influxClient, err := stats.CreateClient()
	defer influxClient.Close()
	ok(t, err)

	dropTable(influxClient, "sessionStatus")
	stats.WriteBatchPoints(influxClient)
	rowCount := getRowCount(influxClient, "sessionStatus")

	assert(t, rowCount > 0, "")
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

	rowCount := getRowCount(influxClient, "provisioningTime")
	assert(t, rowCount > 0, "")
}
