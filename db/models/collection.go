package db

type Collection struct {
	Id           int32  `protobuf:"varint,1,opt,name=Id,proto3" json:"Id,omitempty" yaml:"id,omitempty"`
	Title        string `protobuf:"bytes,2,opt,name=Title,proto3" json:"Title,omitempty" yaml:"title,omitempty"`
	Image        string `protobuf:"bytes,3,opt,name=Image,proto3" json:"Image,omitempty" yaml:"image,omitempty"`
	Website      string `protobuf:"bytes,4,opt,name=Website,proto3" json:"Website,omitempty" yaml:"website,omitempty"`
	ContactEmail string `protobuf:"bytes,5,opt,name=ContactEmail,proto3" json:"ContactEmail,omitempty" yaml:"contactEmail,omitempty"`
	// Why should users view your collection?
	BriefDescription string `protobuf:"bytes,6,opt,name=BriefDescription,proto3" json:"BriefDescription,omitempty" yaml:"briefDescription,omitempty"`
	// Why should users continue and view your lessons?
	LongDescription string           `protobuf:"bytes,7,opt,name=LongDescription,proto3" json:"LongDescription,omitempty" yaml:"longDescription,omitempty"`
	Type            string           `protobuf:"bytes,8,opt,name=Type,proto3" json:"Type,omitempty" yaml:"type,omitempty"`
	Tier            string           `protobuf:"bytes,9,opt,name=Tier,proto3" json:"Tier,omitempty" yaml:"tier,omitempty"`
	CollectionFile  string           `protobuf:"bytes,10,opt,name=CollectionFile,proto3" json:"CollectionFile,omitempty" yaml:"collectionFile,omitempty"`
	Lessons         []*LessonSummary `protobuf:"bytes,11,rep,name=Lessons,proto3" json:"Lessons,omitempty" yaml:"lessons,omitempty"`
}

type LessonSummary struct {
	LessonId          int32  `protobuf:"varint,1,opt,name=lessonId,proto3" json:"lessonId,omitempty" yaml:"lessonId,omitempty"`
	LessonName        string `protobuf:"bytes,2,opt,name=lessonName,proto3" json:"lessonName,omitempty" yaml:"lessonName,omitempty"`
	LessonDescription string `protobuf:"bytes,3,opt,name=lessonDescription,proto3" json:"lessonDescription,omitempty" yaml:"lessonDescription,omitempty"`
}
