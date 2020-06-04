package api

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	config "github.com/nre-learning/antidote-core/config"
	db "github.com/nre-learning/antidote-core/db"
	ingestors "github.com/nre-learning/antidote-core/db/ingestors"
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

func createFakeAPIServer() *AntidoteAPI {
	cfg, err := config.LoadConfig("../../hack/mocks/mock-config-1.yml")
	if err != nil {
		panic(err)
	}
	cfg.CurriculumDir = "../../hack/mocks"

	// Initialize DataManager
	adb := db.NewADMInMem()
	err = ingestors.ImportCurriculum(adb, cfg)
	if err != nil {
		panic(err)
	}

	// Start API server
	lessonAPIServer := AntidoteAPI{
		Config: cfg,
		Db:     adb,
	}

	return &lessonAPIServer
}
