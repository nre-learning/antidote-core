package db

type Curriculum struct {
	Name                 string                `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty" yaml:"name,omitempty"`
	Description          string                `protobuf:"bytes,2,opt,name=Description,proto3" json:"Description,omitempty" yaml:"description,omitempty"`
	Website              string                `protobuf:"bytes,3,opt,name=Website,proto3" json:"Website,omitempty" yaml:"website,omitempty"`
	Lessons              map[int32]*Lesson     `protobuf:"bytes,4,rep,name=Lessons,proto3" json:"Lessons,omitempty" yaml:"lessons,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Collections          map[int32]*Collection `protobuf:"bytes,5,rep,name=Collections,proto3" json:"Collections,omitempty" yaml:"collections,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}