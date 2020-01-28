package db

// Session represents a single provisioned session for interacting with Antidote.
// Sessions hold a one-to-many relationship with LiveLessons - any number of LiveLessons
// can refer to a single Session ID.
type Session struct {
	ID      string `json:"LiveLessonId"`
	ReqAddr string `json:"RequestingAddress"`
}
