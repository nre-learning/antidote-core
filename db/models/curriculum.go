package db

// Curriculum is a resource type that defines a bit of meta-data for a Curriculum as a whole.
type Curriculum struct {
	Slug        string `json:"Slug" yaml:"slug" jsonschema:"Name of this curriculum"`
	Description string `json:"Description" yaml:"description" jsonschema:"Description of this curriculum"`
	Website     string `json:"Website" yaml:"website" jsonschema:"Website for this curriculum"`
	AVer        string `json:"AVer" yaml:"aVer" jsonschema:"The version of Antidote this curriculum was built for"`
}
