package scheduler

import (
	"testing"
)

// testSomething tests something
func testSomething(t *testing.T) {
	kl := &KubeLab{}

	kl.ToLiveLesson()

	t.Fatal("Foobar")
}
