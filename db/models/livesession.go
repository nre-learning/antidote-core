package db

import "time"

// LiveSession represents a single provisioned session for interacting with Antidote.
// LiveSessions hold a one-to-many relationship with LiveLessons - any number of LiveLessons
// can refer to a single LiveSession ID.
type LiveSession struct {
	ID string `json:"id"`

	SourceIP string `json:"sourceIP"`

	// This replaces the old whitelist model. To preserve a session through GC, set this to true.
	// Set to false by default on creation
	Persistent bool `json:"persistent"`

	CreatedTime time.Time `json:"CreatedTime"`
}
