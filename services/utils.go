package services

import "fmt"

// Because there could be multiple instances of Antidote running on some shared infrastructure (e.g. the same Kubernetes cluster),
// it's important to be able to create a truly unique identifier for LiveLessons, which include not only the LiveLesson ID but also the Antidote instance ID.
// It's also important that these IDs are generated from the same source, to ensure they match, since some backends may use them in certain ways (i.e. Kubernetes namespace names)
// but the API also uses these to disambiguate HEPS subdomains. This struct and its associated functions are meant to be used to ensure this is the case.
type UULLID struct {
	AntidoteID   string
	LiveLessonID string
}

func NewUULLID(antidoteID, liveLessonID string) UULLID {
	return UULLID{AntidoteID: antidoteID, LiveLessonID: liveLessonID}
}

func (u UULLID) ToString() string {
	return fmt.Sprintf("%s-%s", u.AntidoteID, u.LiveLessonID)
}
