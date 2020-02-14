package db

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
