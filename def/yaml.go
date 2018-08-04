package def

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type LabDefinition struct {
	LabName        string        `json:"labName" yaml:"labName"`
	LabID          int32         `json:"labID" yaml:"labID"`
	Devices        []*Device     `json:"devices" yaml:"devices"`
	Connections    []*Connection `json:"connections" yaml:"connections"`
	SharedTopology bool          `json:"SharedTopology" yaml:"sharedTopology"`
	Notebook       bool          `json:"notebook" yaml:"notebook"`
	LabGuide       string        `json:"labguide" yaml:"labguide"`
	Category       string        `json:"category" yaml:"category"`
}

func (ld *LabDefinition) Json() string {
	labJson, err := json.Marshal(ld)
	if err != nil {
		log.Error(err)
		return ""
	}

	return string(labJson)
}

type Device struct {
	Name  string `json:"name" yaml:"name"`
	Image string `json:"image" yaml:"image"`
}

type Connection struct {
	A string `json:"a" yaml:"a"`
	B string `json:"b" yaml:"b"`
}

func ImportLabDefs(fileList []string) (map[int32]*LabDefinition, error) {

	retLds := map[int32]*LabDefinition{}

FILES:
	for f := range fileList {

		file := fileList[f]

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			return map[int32]*LabDefinition{}, err
		}

		var labDef LabDefinition
		err = yaml.Unmarshal([]byte(yamlDef), &labDef)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
			continue FILES
		}

		if labDef.LabName == "" {
			log.Errorf("Failed to import %s: %s", file, errors.New("Lab name cannot be blank"))
			continue FILES
		}

		if labDef.LabID == 0 {
			log.Info(labDef.Json())
			log.Errorf("Failed to import %s: %s", file, errors.New("Lab id cannot be 0"))
			continue FILES
		}

		if !labDef.SharedTopology {
			if len(labDef.Devices) == 0 {
				log.Errorf("Failed to import %s: %s", file, errors.New("Devices list is empty and sharedTopology is set to false"))
				continue FILES
			}
			if len(labDef.Connections) == 0 {
				log.Errorf("Failed to import %s: %s", file, errors.New("Connections list is empty and sharedTopology is set to false"))
				continue FILES
			}
		}

		for i := range labDef.Devices {
			device := labDef.Devices[i]

			if device.Name == "" {
				log.Errorf("Failed to import %s: %s", file, errors.New("Device name cannot be blank"))
				continue FILES
			}
			if device.Image == "" {
				log.Errorf("Failed to import %s: %s", file, errors.New("Device image cannot be blank"))
				continue FILES
			}
		}

		for c := range labDef.Connections {
			connection := labDef.Connections[c]

			if !deviceInLabDef(connection.A, &labDef) {
				log.Errorf("Failed to import %s: %s", file, errors.New("Connection refers to nonexistent device"))
				continue FILES
			}

			if !deviceInLabDef(connection.B, &labDef) {
				log.Errorf("Failed to import %s: %s", file, errors.New("Connection refers to nonexistent device"))
				continue FILES
			}
		}

		// TODO(mierdin): Make sure lab ID and lab name are unique

		// TODO(mierdin): Make sure notebook exists as "lesson.ipynb" adjacent to the definition file if set to true

		log.Infof("Successfully imported %s: %v", file, labDef)

		retLds[labDef.LabID] = &labDef
	}

	return retLds, nil
}

// deviceInLabDef is a helper function to ensure that a device is found by name in a lab definition
func deviceInLabDef(deviceName string, ld *LabDefinition) bool {
	for i := range ld.Devices {
		device := ld.Devices[i]
		if deviceName == device.Name {
			return true
		}
	}
	return false
}
