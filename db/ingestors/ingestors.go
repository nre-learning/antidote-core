package db

import "errors"

var (
	tierMap = map[string]int{
		"prod":  3,
		"ptr":   2,
		"local": 1,
	}

	// General Errors
	errBasicValidation = errors.New("Failed basic validation")
	errTierMismatch    = errors.New("Lesson has a lower than desired tier")

	// Lessons-Specific Errors
	errInsufficientPresentation = errors.New("No presentations configured, and no additionalPorts specified")
	errMissingConfigurationFile = errors.New("Configuration file not found")
	errDuplicateEndpoint        = errors.New("Duplicate endpoints detected")
	errDuplicatePresentation    = errors.New("Duplicate presentations detected")
	errBadConnection            = errors.New("Malformed connection")
	errMissingLessonGuide       = errors.New("Couldn't find/read lesson guide")
)
