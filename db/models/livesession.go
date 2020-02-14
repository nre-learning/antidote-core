package db

type LiveSession struct {
	ID string `json:"id"`

	SourceIP string `json:"sourceIP"`

	// This replaces the old whitelist model. To preserve a session through GC, set this to true.
	// Set to false by default on creation
	Persistent bool `json: "persistent"`
}
