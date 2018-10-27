package def

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	config "github.com/nre-learning/syringe/config"
	"gopkg.in/yaml.v2"
)

type LessonDefinition struct {
	LessonName    string                 `json:"lessonName" yaml:"lessonName"`
	LessonID      int32                  `json:"lessonID" yaml:"lessonID"`
	Disabled      bool                   `json:"disabled" yaml:"disabled"`
	Devices       []*Endpoint            `json:"devices" yaml:"devices"`
	Utilities     []*Endpoint            `json:"utilities" yaml:"utilities"`
	Blackboxes    []*Endpoint            `json:"blackboxes" yaml:"blackboxes"`
	Connections   []*Connection          `json:"connections" yaml:"connections"`
	TopologyType  string                 `json:"topologyType" yaml:"topologyType"`
	Stages        map[int32]*LessonStage `json:"stages" yaml:"stages"`
	Category      string                 `json:"category" yaml:"category"`
	LessonDiagram string                 `json:"lessondiagram" yaml:"lessondiagram"`
	LessonVideo   string                 `json:"lessonvideo" yaml:"lessonvideo"`
	Tier          string                 `json:"tier" yaml:"tier"`
}

type Endpoint struct {
	Name        string  `json:"name" yaml:"name"`
	Image       string  `json:"image" yaml:"image"`
	Sshuser     string  `json:"sshuser" yaml:"sshuser"`
	Sshpassword string  `json:"sshpassword" yaml:"sshpassword"`
	Ports       []int32 `json:"ports" yaml:"ports"`
}

type LessonStage struct {
	LabGuide       string            `json:"labguide" yaml:"labguide"`
	Configs        map[string]string `json:"configs" yaml:"configs"`
	IframeResource IframeDetails     `json:"iframeresource" yaml:"iframeresource"`
	Description    string            `json:"description" yaml:"description"`
}

type IframeDetails struct {
	Name     string `json:"name" yaml:"name"`
	Protocol string `json:"protocol" yaml:"protocol"`
	URI      string `json:"uri" yaml:"uri"`
	Port     int32  `json:"port" yaml:"port"`
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

func ImportLessonDefs(syringeConfig *config.SyringeConfig, fileList []string) (map[int32]*LessonDefinition, error) {

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

		if lessonDef.Tier == "" {
			log.Errorf("Failed to import %s: %s", file, errors.New("Must provide tier"))
			continue FILES
		}

		if lessonDef.Tier != "local" && lessonDef.Tier != "ptr" && lessonDef.Tier != "prod" {
			log.Errorf("Failed to import %s: %s", file, errors.New("Invalid tier value"))
			continue FILES
		}

		if lessonDef.Tier == "ptr" && syringeConfig.Tier == "local" {
			continue FILES
		}
		if lessonDef.Tier == "local" && syringeConfig.Tier != "local" {
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
				log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Connection %s refers to nonexistent entity", connection.A)))
				continue FILES
			}

			if !entityInLabDef(connection.B, &lessonDef) {
				log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Connection %s refers to nonexistent entity", connection.B)))
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
	for i := range ld.Blackboxes {
		blackbox := ld.Blackboxes[i]
		if entityName == blackbox.Name {
			return true
		}
	}
	return false
}
