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
	os.Setenv("SYRINGE_LESSONS", "foo")
	os.Setenv("SYRINGE_DOMAIN", "bar")
	syringeConfig, err := LoadConfigVars()
	if err != nil {
		t.Fatal(err)
	}

	// Pretty barbaric but works for now
	assert(t, syringeConfig.JSON() == `{"LessonsDir":"foo","Tier":"local","Domain":"bar","GRPCPort":50099,"HTTPPort":8086,"DeviceGCAge":0,"NonDeviceGCAge":0,"HealthCheckInterval":0,"TSDBExportInterval":0,"TSDBEnabled":false,"LessonTTL":30,"LessonsLocal":false,"LessonRepoRemote":"https://github.com/nre-learning/antidote.git","LessonRepoBranch":"master","LessonRepoDir":"/antidote"}`, "")
}
