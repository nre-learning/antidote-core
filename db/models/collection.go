package db

// Collection is a resource type that provides a type of categorization for other curriculum resources
// like Lessons. A Collection might be defined for a company, an open-source project, or even for an individual,
// as a home for all curriculum resources with strong relationships to that entity, and as a way of giving
// more information for that entity.
type Collection struct {
	Slug             string   `json:"Id,omitempty" yaml:"id,omitempty"`
	Title            string   `json:"Title,omitempty" yaml:"title,omitempty"`
	Image            string   `json:"Image,omitempty" yaml:"image,omitempty"`
	Website          string   `json:"Website,omitempty" yaml:"website,omitempty"`
	ContactEmail     string   `json:"ContactEmail,omitempty" yaml:"contactEmail,omitempty"`
	BriefDescription string   `json:"BriefDescription,omitempty" yaml:"briefDescription,omitempty"`
	LongDescription  string   `json:"LongDescription,omitempty" yaml:"longDescription,omitempty"`
	Type             string   `json:"Type,omitempty" yaml:"type,omitempty"`
	Tier             string   `json:"Tier,omitempty" yaml:"tier,omitempty"`
	CollectionFile   string   `json:"CollectionFile,omitempty" yaml:"collectionFile,omitempty"`
	Lessons          []string `json:"Lessons,omitempty" yaml:"lessons,omitempty"`
}
