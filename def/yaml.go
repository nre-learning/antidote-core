package def

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type LessonDefinition struct {
	LessonName    string                 `json:"lessonName" yaml:"lessonName"`
	LessonID      int32                  `json:"lessonID" yaml:"lessonID"`
	Devices       []*Device              `json:"devices" yaml:"devices"`
	Utilities     []*Utility             `json:"utilities" yaml:"utilities"`
	Connections   []*Connection          `json:"connections" yaml:"connections"`
	TopologyType  string                 `json:"topologyType" yaml:"topologyType"`
	Stages        map[int32]*LessonStage `json:"stages" yaml:"stages"`
	Notebook      bool                   `json:"notebook" yaml:"notebook"`
	Category      string                 `json:"category" yaml:"category"`
	LessonDiagram string                 `json:"lessondiagram" yaml:"lessondiagram"`
}

type LessonStage struct {
	LabGuide    string            `json:"labguide" yaml:"labguide"`
	Configs     map[string]string `json:"configs" yaml:"configs"`
	Notebook    bool              `json:"notebook" yaml:"notebook"`
	Description string            `json:"description" yaml:"description"`
}

type Device struct {
	Name  string `json:"name" yaml:"name"`
	Image string `json:"image" yaml:"image"`
}

type Utility struct {
	Name  string `json:"name" yaml:"name"`
	Image string `json:"image" yaml:"image"`
}

type Connection struct {
	A      string `json:"a" yaml:"a"`
	B      string `json:"b" yaml:"b"`
	Subnet string `json:"subnet" yaml:"subnet"`
}

// JSON exports the lesson definition as JSON
func (ld *LessonDefinition) JSON() string {
	lessonJSON, err := json.MarshalIndent(ld, "", "  ")
	if err != nil {
		log.Error(err)
		return ""
	}

	return string(lessonJSON)
}

func ImportLessonDefs(fileList []string) (map[int32]*LessonDefinition, error) {

	retLds := map[int32]*LessonDefinition{}

FILES:
	for f := range fileList {

		file := fileList[f]

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue FILES
		}

		var lessonDef LessonDefinition
		err = yaml.Unmarshal([]byte(yamlDef), &lessonDef)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
			continue FILES
		}

		if lessonDef.LessonName == "" {
			log.Errorf("Failed to import %s: %s", file, errors.New("Lesson name cannot be blank"))
			continue FILES
		}

		if lessonDef.Category == "" {
			log.Errorf("Failed to import %s: %s", file, errors.New("Lesson category cannot be blank"))
			continue FILES
		}

		if lessonDef.LessonID == 0 {
			log.Info(lessonDef.JSON())
			log.Errorf("Failed to import %s: %s", file, errors.New("Lesson id cannot be 0"))
			continue FILES
		}

		if lessonDef.TopologyType != "none" && lessonDef.TopologyType != "shared" && lessonDef.TopologyType != "custom" {
			lessonDef.TopologyType = "none"
		}

		if lessonDef.TopologyType == "custom" {
			if len(lessonDef.Devices) == 0 {
				log.Errorf("Failed to import %s: %s", file, errors.New("Devices list is empty and TopologyType is set to custom"))
				continue FILES
			}
			if len(lessonDef.Connections) == 0 {
				log.Errorf("Failed to import %s: %s", file, errors.New("Connections list is empty and TopologyType is set to custom"))
				continue FILES
			}
		}

		for i := range lessonDef.Devices {
			device := lessonDef.Devices[i]

			if device.Name == "" {
				log.Errorf("Failed to import %s: %s", file, errors.New("Device name cannot be blank"))
				continue FILES
			}
			if device.Image == "" {
				log.Errorf("Failed to import %s: %s", file, errors.New("Device image cannot be blank"))
				continue FILES
			}
		}

		for c := range lessonDef.Connections {
			connection := lessonDef.Connections[c]

			if !entityInLabDef(connection.A, &lessonDef) {
				log.Errorf("Failed to import %s: %s", file, errors.New("Connection refers to nonexistent entity"))
				continue FILES
			}

			if !entityInLabDef(connection.B, &lessonDef) {
				log.Errorf("Failed to import %s: %s", file, errors.New("Connection refers to nonexistent entity"))
				continue FILES
			}

			if connection.Subnet == "" {
				log.Errorf("Connection must specify subnet")
				continue FILES
			}
		}

		// TODO(mierdin): Make sure lab ID and lab name are unique

		// log.Infof("Successfully imported %s: %v", file, lessonDef.JSON())
		log.Infof("Successfully imported lesson %d: %s", lessonDef.LessonID, lessonDef.LessonName)

		retLds[lessonDef.LessonID] = &lessonDef
	}

	return retLds, nil
}

// entityInLabDef is a helper function to ensure that a device is found by name in a lab definition
func entityInLabDef(entityName string, ld *LessonDefinition) bool {
	for i := range ld.Devices {
		device := ld.Devices[i]
		if entityName == device.Name {
			return true
		}
	}
	for i := range ld.Utilities {
		utility := ld.Utilities[i]
		if entityName == utility.Name {
			return true
		}
	}
	return false
}
