package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// Helper functions courtesy of the venerable Ben Johnson
// https://medium.com/@benbjohnson/structuring-tests-in-go-46ddee7a25c

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// TestConfigJSON ensures a given config renders correctly as JSON
func TestConfigJSON(t *testing.T) {
	os.Setenv("SYRINGE_CURRICULUM", "foo")
	os.Setenv("SYRINGE_DOMAIN", "bar")
	syringeConfig, err := LoadConfigVars()
	if err != nil {
		t.Fatal(err)
	}

	desired := SyringeConfig{
		CurriculumDir:        "foo",
		Tier:                 "local",
		Domain:               "bar",
		GRPCPort:             50099,
		HTTPPort:             8086,
		DeviceGCAge:          0,
		NonDeviceGCAge:       0,
		HealthCheckInterval:  0,
		LiveLessonTTL:        30,
		InfluxURL:            "https://influxdb.networkreliability.engineering/",
		InfluxUsername:       "admin",
		InfluxPassword:       "zerocool",
		TSDBExportInterval:   0,
		TSDBEnabled:          false,
		CurriculumLocal:      false,
		CurriculumVersion:    "latest",
		CurriculumRepoRemote: "https://github.com/nre-learning/nrelabs-curriculum.git",
		CurriculumRepoBranch: "master",
		PrivilegedImages: []string{
			"antidotelabs/container-vqfx",
			"antidotelabs/vqfx-snap1",
			"antidotelabs/vqfx-snap2",
			"antidotelabs/vqfx-snap3",
			"antidotelabs/vqfx-full",
			"antidotelabs/cvx-3.7.6",
			"antidotelabs/frr-7.1",
		},
		AllowEgress: false,
	}

	t.Log(syringeConfig.JSON())
	t.Log(desired.JSON())

	// Pretty barbaric but works for now
	assert(t, syringeConfig.JSON() == desired.JSON(), "")
}
