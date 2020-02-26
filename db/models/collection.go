package db

// Collection is a resource type that provides a type of categorization for other curriculum resources
// like Lessons. A Collection might be defined for a company, an open-source project, or even for an individual,
// as a home for all curriculum resources with strong relationships to that entity, and as a way of giving
// more information for that entity.
type Collection struct {
	Slug             string   `json:"Id" yaml:"id"`
	Title            string   `json:"Title" yaml:"title"`
	Image            string   `json:"Image" yaml:"image"`
	Website          string   `json:"Website" yaml:"website"`
	ContactEmail     string   `json:"ContactEmail" yaml:"contactEmail"`
	BriefDescription string   `json:"BriefDescription" yaml:"briefDescription"`
	LongDescription  string   `json:"LongDescription" yaml:"longDescription"`
	Type             string   `json:"Type" yaml:"type"`
	Tier             string   `json:"Tier" yaml:"tier"`
	CollectionFile   string   `json:"CollectionFile" yaml:"collectionFile"`
	Lessons          []string `json:"Lessons" yaml:"lessons"`
}
