package api

import (
	"context"
	"testing"

	pb "github.com/nre-learning/antidote-core/api/exp/generated"
	ot "github.com/opentracing/opentracing-go"
	// Fake clients
)

func TestUpdateLiveSessionPersistence(t *testing.T) {
	span := ot.StartSpan("test_api_session_persistence")
	defer span.Finish()

	api := createFakeAPIServer()

	ctx := context.Background()

	_, err := api.CreateLiveSession(ctx, &pb.LiveSession{ID: "abcdef", Persistent: false})
	ok(t, err)

	ls, err := api.GetLiveSession(ctx, &pb.LiveSession{ID: "abcdef"})

	// Assert session record has been created with correct values
	// for ID and Persistent
	ok(t, err)
	assert(t, (ls != nil), "")
	assert(t, (ls.ID == "abcdef"), "")
	equals(t, (ls.Persistent == false), true)

	_, err = api.UpdateLiveSessionPersistence(ctx, &pb.SessionPersistence{SessionID: "abcdef", Persistent: true})
	ok(t, err)

	ls, err = api.GetLiveSession(ctx, &pb.LiveSession{ID: "abcdef"})

	// Assert Persistent value in the session record has been updated
	ok(t, err)
	assert(t, (ls.ID == "abcdef"), "")
	equals(t, (ls.Persistent == true), true)
}
