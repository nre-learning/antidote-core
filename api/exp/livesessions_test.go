package api

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	ot "github.com/opentracing/opentracing-go"
	// Fake clients
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

func TestUpdateLiveSessionPersistence(t *testing.T) {
	span := ot.StartSpan("test_api_session_persistence")
	defer span.Finish()

	api := createFakeAPIServer()

	ctx := context.Background()

	_, err := api.CreateLiveSession(ctx, &pb.LiveSession{ID: "abcdef", Persistent: false})
	ok(t, err)

	ls, err := api.GetLiveSession(ctx, &pb.LiveSession{ID: "abcdef"})

	// Assert namespace exists without error
	ok(t, err)
	assert(t, (ls != nil), "")

	_, err = api.UpdateLiveSessionPersistence(ctx, &pb.SessionPersistence{SessionID: "abcdef", Persistent: true})
	ok(t, err)

	ls, err = api.GetLiveSession(ctx, &pb.LiveSession{ID: "abcdef"})

	// Assert namespace exists without error
	ok(t, err)
	equals(t, (ls.Persistent == true), true)
}
