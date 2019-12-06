package db

// CurriculumResource is a specific type of database model that is designed to be imported from a YAML file.
// Only these types of resources require a JSON schema for validation purposes, so as a result, we can identify
// them by their inclusion of relevant schema validation function(s)
//
// Database models that do not satisfy this interface are used for other purposes, such as state tracking, etc.
type CurriculumResource interface {
	JSValidate() bool
}

// EnforceInterfaceCompliance uses CurriculumResource types to ensure conformance with the interface
func EnforceInterfaceCompliance() {
	func(cr CurriculumResource) {}(Lesson{})
}
