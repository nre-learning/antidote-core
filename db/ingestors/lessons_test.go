package db

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
)

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

// getValidLesson returns a full, valid example of a Lesson that uses all the features.
// Tests in this file should make use of this by making a copy, tweaking in some way that makes it
// invalid, and then asserting on the error type/message.
func getValidLesson() models.Lesson {

	lessons, err := ReadLessons(config.AntidoteConfig{
		CurriculumDir: "../test/test-curriculum",
		Tier:          "local",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", lessons)
	return *lessons[0]
}

func TestValidLesson(t *testing.T) {
	l := getValidLesson()
	err := validateLesson(&l)
	assert(t, (err == nil), "Expected validation to pass, but encountered validation errors")
}

func TestInvalidCharInImageName(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Image = "antidotelabs/utility:latest"
	err := validateLesson(&l)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}

// All Presentations must specify a nonzero TCP port
func TestMissingPresentationPort(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Presentations[0].Port = 0
	err := validateLesson(&l)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}
