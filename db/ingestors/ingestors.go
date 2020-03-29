package db

import "errors"

var (
	// General
	errBasicValidation = errors.New("Failed basic validation")
	errTierMismatch    = errors.New("Lesson has a lower than desired tier")

	// Lessons
	errInsufficientPresentation = errors.New("No presentations configured, and no additionalPorts specified")
	errMissingConfigurationFile = errors.New("Configuration file not found")
	errDuplicatePresentation    = errors.New("Duplicate presentations detected")
	errBadConnection            = errors.New("Malformed connection")
	errMissingLessonGuide       = errors.New("Couldn't find/read lesson guide")
)
