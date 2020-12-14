package db

import (
	"encoding/json"

	jsonschema "github.com/alecthomas/jsonschema"
	log "github.com/sirupsen/logrus"
	gjs "github.com/xeipuuv/gojsonschema"
)

// Lesson represents the fields and sub-types for defining a lesson resource in Antidote
type Lesson struct {
	Name             string              `json:"Name" yaml:"name" jsonschema:"minLength=1,description=Human-readable name/title for the lesson"`
	Slug             string              `json:"Slug" yaml:"slug" jsonschema:"minLength=1,description=A unique identifier for the lesson (usually 2-3 lower-case words with hyphens),pattern=^[a-z-]*$"`
	Category         string              `json:"Category" yaml:"category" jsonschema:"minLength=1,description=The name for the Category in which this lesson should belong,enum=fundamentals,enum=tools,enum=workflows"`
	Diagram          string              `json:"Diagram" yaml:"diagram" jsonschema:"description=A public URL to lesson diagram"`
	Video            string              `json:"Video" yaml:"video" jsonschema:"description=YouTube URL to lesson video"`
	Tier             string              `json:"Tier" yaml:"tier" jsonschema:"minLength=1,description=Tier for this lesson (you probably want 'prod') ,enum=prod,enum=ptr,enum=local"`
	Collection       string              `json:"Collection,omitempty" yaml:"collection,omitempty" jsonschema:"description=The slug for the collection this lesson should belong to"`
	Description      string              `json:"Description" yaml:"description" jsonschema:"minLength=1,description=A helpful description for what the learner should expect to get from this lesson"`
	ShortDescription string              `json:"ShortDescription" yaml:"shortDescription" jsonschema:"minLength=1,description=A brief description for this lesson. One or two words"`
	Prereqs          []string            `json:"Prereqs,omitempty" yaml:"prereqs,omitempty" jsonschema:"description=A list of slugs for other lessons that are prerequisite to this lesson"`
	Tags             []string            `json:"Tags,omitempty" yaml:"tags,omitempty" jsonschema:"description=A list of tags to apply to this lesson for categorization purposes"`
	Stages           []*LessonStage      `json:"Stages" yaml:"stages" jsonschema:"minItems=1,description=(Logical sections or chapters of a lesson)\nhttps://docs.nrelabs.io/antidote/object-reference/lessons/stages,additionalProperties=false"`
	Endpoints        []*LessonEndpoint   `json:"Endpoints" yaml:"endpoints" jsonschema:"minItems=1,description=(An instance of a software image to be made available in the lesson)\nhttps://docs.nrelabs.io/antidote/object-reference/lessons/endpoints"`
	Connections      []*LessonConnection `json:"Connections,omitempty" yaml:"connections,omitempty" jsonschema:"description=(Connections between endpoints in the topology)\nhttps://docs.nrelabs.io/antidote/object-reference/lessons/connections"`
	Authors          []*LessonAuthor     `json:"Authors,omitempty" yaml:"authors,omitempty" jsonschema:"description=(A list of individuals that have contributed to this lesson)\nhttps://docs.nrelabs.io/antidote/object-reference/lessons/authors"`

	// Experimental field for adding a short delay before marking the lesson ready, to help deal with some SSH instability for images that have processes still starting.
	ReadyDelay int `json:"-" yaml:"readyDelay,omitempty" jsonschema:"-"`

	// NOTE - any time you see these dashes, it means this field is used for internal purposes only.
	// When we were using protobuf models for everything, we couldn't do this. But by separating internal
	// models from API models, we can still mark a field like this while still being able to offer it via API.
	// Adding a hash to the json tag for instance doesn't mean we're hiding it from the API.
	LessonFile string `json:"-" yaml:"-" jsonschema:"-"`
	LessonDir  string `json:"-" yaml:"-" jsonschema:"-"`
}

// `jsonschema:"additionalProperties=false"`

// LessonStage is a specific state that a Lesson can be in. This can be thought of like chapters in a book.
// A Lesson might have one or more LessonStages.
type LessonStage struct {
	Description   string          `json:"Description,omitempty" yaml:"description,omitempty"`
	GuideType     LessonGuideType `json:"GuideType" yaml:"guideType" jsonschema:"required,enum=markdown,enum=jupyter"`
	GuideContents string          `json:"-"  yaml:"-" jsonschema:"-"`
	StageVideo    string          `json:"StageVideo" yaml:"stageVideo" jsonschema:"description=URL to lesson stage video"`
}

type LessonGuideType string

const (
	GuideMarkdown LessonGuideType = "markdown"
	GuideJupyter  LessonGuideType = "jupyter"
)

// LessonEndpoint is typically a container that runs some software in a Lesson. This can be a network device,
// or a simple container with some Python libraries installed - it doesn't really matter. It's just some software
// that you want to have running in a lesson for educational purposes
type LessonEndpoint struct {
	Name  string `json:"Name" yaml:"name" jsonschema:"description=Name of the endpoint"`
	Image string `json:"Image" yaml:"image" jsonschema:"description=The Image ref this endpoint uses,pattern=^[A-Za-z0-9\\-]*$"`

	ConfigurationType string `json:"ConfigurationType,omitempty" yaml:"configurationType,omitempty" jsonschema:"description=Method for configuring this endpoint at runtime,enum=,enum=napalm,enum=python,enum=ansible"`

	// Since we're starting to use the filename to derive certain things about configuration (i.e.
	// which NAPALM driver to use) we will store the filename (only the name, no path) here on ingest
	// so we know where to look when it comes time to push a configuration
	ConfigurationFile string `json:"-" yaml:"-" jsonschema:"-"`

	AdditionalPorts []int32 `json:"AdditionalPorts,omitempty" yaml:"additionalPorts,omitempty" jsonschema:"description=Additional ports to open that aren't in a Presentation"`

	Presentations []*LessonPresentation `json:"Presentations,omitempty" yaml:"presentations,omitempty" jsonschema:"description=(A mechanism for providing the user with interactive access to this endpoint)\nhttps://docs.nrelabs.io/antidote/object-reference/lessons/presentations"`
}

// LessonPresentation is a particular view into a LessonEndpoint. It's a way of specifying how an endpoint
// should be made available to the user in the front-end
type LessonPresentation struct {
	Name string           `json:"Name" yaml:"name" jsonschema:"required"`
	Port int32            `json:"Port" yaml:"port" jsonschema:"required,minimum=1"`
	Type PresentationType `json:"Type" yaml:"type" jsonschema:"required,enum=http,enum=ssh"`
}

// LessonConnection is a point-to-point network connection between two LessonEndpoints. The `A` and `B` properties should
// refer to the Name of LessonEndpoints within a given lesson that should be networked together.
type LessonConnection struct {
	A      string `json:"A" yaml:"a" jsonschema:"required"`
	B      string `json:"B" yaml:"b" jsonschema:"required"`
	Subnet string `json:"Subnet" yaml:"subnet" jsonschema:"required"`
}

// PresentationType is backed by a set of possible const values for presentation types below
type PresentationType string

const (

	// PresentationType_http is for presentations that use iframes to present a web front-end to the user
	PresentationType_http PresentationType = "http"

	// PresentationType_ssh is for presentations that provide an interactive terminal
	PresentationType_ssh PresentationType = "ssh"
)

// LessonAuthor represents details about an individual who participated in authoring this lesson
type LessonAuthor struct {
	Name string `jsonschema:"description=Full name of this author"`
	Link string `jsonschema:"description=URL to this author's profile such as Twitter or Github"`
}

// GetSchema returns a Schema to be used in creation wizards
func (l Lesson) GetSchema() *jsonschema.Schema {
	return jsonschema.Reflect(l)
}

// JSValidate uses an Antidote resource's struct properties and tags to construct a jsonschema
// document, and then validates that instance's values against that schema.
func (l Lesson) JSValidate() bool {

	// Load JSON Schema document for Lesson type
	schemaReflect := jsonschema.Reflect(l)
	b, _ := json.Marshal(schemaReflect)
	schemaLoader := gjs.NewStringLoader(string(b))
	schema, _ := gjs.NewSchema(schemaLoader)

	// Load lesson instance JSON document
	b, err := json.Marshal(l)
	if err != nil {
		log.Error(err)
		return false
	}
	documentLoader := gjs.NewStringLoader(string(b))

	// Perform validation
	result, err := schema.Validate(documentLoader)
	if err != nil {
		log.Error(err)
		return false
	}

	validationErrors := result.Errors()
	for i := range validationErrors {
		log.Errorf("Validation error - %s", validationErrors[i].String())
	}

	return result.Valid()
}
