package db

import jsonschema "github.com/alecthomas/jsonschema"

// CurriculumResource is a specific type of database model that is designed to be imported from a YAML file.
// Only these types of resources require a JSON schema for validation purposes, so as a result, we can identify
// them by their inclusion of relevant schema validation function(s)
//
// Database models that do not satisfy this interface are used for other purposes, such as state tracking, etc.
type CurriculumResource interface {
	JSValidate() bool
	GetSchema() *jsonschema.Schema
}

var _ CurriculumResource = (*Lesson)(nil)

// var _ CurriculumResource = (*Collection)(nil)

// UsesJupyterLabGuide is a helper function that lets us know if a lesson def uses a
// jupyter notebook as a lab guide in any stage.
func UsesJupyterLabGuide(lesson Lesson) bool {

	for i := range lesson.Stages {
		if lesson.Stages[i].GuideType == GuideJupyter {
			return true
		}
	}

	return false
}
