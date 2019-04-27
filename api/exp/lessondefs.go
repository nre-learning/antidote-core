package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func (s *SyringeAPIServer) ListLessonDefs(ctx context.Context, filter *pb.LessonDefFilter) (*pb.LessonDefs, error) {

	defs := []*pb.LessonDef{}

	// TODO(mierdin): Okay for now, but not super efficient. Should store in category keys when loaded.
	for _, lessonDef := range s.Scheduler.LessonDefs {

		if filter.Category == "" {
			defs = append(defs, lessonDef)
			continue
		}

		if lessonDef.Category == filter.Category {
			defs = append(defs, lessonDef)
		}
	}

	return &pb.LessonDefs{
		LessonDefs: defs,
	}, nil
}

// var preReqs []int32

func (s *SyringeAPIServer) GetAllLessonPrereqs(ctx context.Context, lid *pb.LessonID) (*pb.LessonPrereqs, error) {

	// Preload the requested lesson ID so we can strip it before returning
	pr := s.getPrereqs(lid.Id, []int32{lid.Id})
	log.Debugf("Getting prerequisites for Lesson %d: %d", lid.Id, pr)

	return &pb.LessonPrereqs{
		// Remove first item from slice - this is the lesson ID being requested
		Prereqs: pr[1:],
	}, nil
}

func (s *SyringeAPIServer) getPrereqs(lessonID int32, currentPrereqs []int32) []int32 {

	// Return if lesson ID doesn't exist
	if _, ok := s.Scheduler.LessonDefs[lessonID]; !ok {
		return currentPrereqs
	}

	// Add this lessonID to prereqs if doesn't already exist
	if !isAlreadyInSlice(lessonID, currentPrereqs) {
		currentPrereqs = append(currentPrereqs, lessonID)
	}

	// Return if lesson doesn't have prerequisites
	lesson := s.Scheduler.LessonDefs[lessonID]
	if len(lesson.Prereqs) == 0 {
		return currentPrereqs
	}

	// Call recursion for lesson IDs that need it
	for i := range lesson.Prereqs {
		pid := lesson.Prereqs[i]
		currentPrereqs = s.getPrereqs(pid, currentPrereqs)
	}

	return currentPrereqs
}

func isAlreadyInSlice(lessonID int32, currentPrereqs []int32) bool {
	for i := range currentPrereqs {
		if currentPrereqs[i] == lessonID {
			return true
		}
	}
	return false
}

func (s *SyringeAPIServer) GetLessonDef(ctx context.Context, lid *pb.LessonID) (*pb.LessonDef, error) {

	if _, ok := s.Scheduler.LessonDefs[lid.Id]; !ok {
		return nil, errors.New("Invalid lesson ID")
	}

	lessonDef := s.Scheduler.LessonDefs[lid.Id]

	return lessonDef, nil
}

func ImportLessonDefs(syringeConfig *config.SyringeConfig) (map[int32]*pb.LessonDef, error) {

	// Get lesson definitions
	fileList := []string{}
	lessonDir := fmt.Sprintf("%s/lessons", syringeConfig.CurriculumDir)
	log.Debugf("Searching %s for lesson definitions", lessonDir)
	err := filepath.Walk(lessonDir, func(path string, f os.FileInfo, err error) error {
		syringeFileLocation := fmt.Sprintf("%s/lesson.meta.yaml", path)
		if _, err := os.Stat(syringeFileLocation); err == nil {
			log.Debugf("Found lesson definition at: %s", syringeFileLocation)
			fileList = append(fileList, syringeFileLocation)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	retLds := map[int32]*pb.LessonDef{}

	for f := range fileList {

		file := fileList[f]

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue
		}

		var lessonDef pb.LessonDef
		err = yaml.Unmarshal([]byte(yamlDef), &lessonDef)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
		}
		lessonDef.LessonFile = file

		// Set type property as appropriate
		for ep := range lessonDef.Blackboxes {
			lessonDef.Blackboxes[ep].Type = pb.Endpoint_BLACKBOX
		}
		for ep := range lessonDef.Utilities {
			lessonDef.Utilities[ep].Type = pb.Endpoint_UTILITY
		}
		for ep := range lessonDef.Devices {
			lessonDef.Devices[ep].Type = pb.Endpoint_DEVICE
		}

		if _, ok := retLds[lessonDef.LessonId]; ok {
			log.Errorf("Failed to import %s: Lesson ID %d already exists in another lesson definition.", file, lessonDef.LessonId)
			continue
		}

		err = validateLesson(syringeConfig, &lessonDef)
		if err != nil {
			continue
		}
		log.Infof("Successfully imported lesson %d: %s --- BLACKBOX: %d, IFR: %d, UTILITY: %d, DEVICE: %d, CONNECTIONS: %d", lessonDef.LessonId, lessonDef.LessonName,
			len(lessonDef.Blackboxes),
			len(lessonDef.IframeResources),
			len(lessonDef.Utilities),
			len(lessonDef.Devices),
			len(lessonDef.Connections),
		)

		// Insert stage at zero-index so we can use slice indexes to refer to each stage without jumping through hoops
		// or making the user use 0 as a stage ID
		lessonDef.Stages = append([]*pb.LessonStage{{Id: 0}}, lessonDef.Stages...)

		retLds[lessonDef.LessonId] = &lessonDef
	}

	if len(fileList) == len(retLds) {
		log.Infof("Imported %d lesson definitions.", len(retLds))
		return retLds, nil
	} else {
		log.Warnf("Imported %d lesson definitions with errors.", len(retLds))
		return retLds, errors.New("Not all lesson definitions were imported")
	}

}

// validateLesson validates a single lesson, returning a simple error if the lesson fails
// to validate.
func validateLesson(syringeConfig *config.SyringeConfig, lessonDef *pb.LessonDef) error {

	// TODO(mierdin): In the future, you should consider putting unique error messages for
	// each violation. This will make this function more testable.
	fail := errors.New("failed to validate lesson definition")

	file := lessonDef.LessonFile

	// Basic validation from protobuf tags
	err := lessonDef.Validate()
	if err != nil {
		log.Errorf("Basic validation failed on %s: %s", file, err)
		return fail
	}

	// More advanced validation
	if syringeConfig.Tier == "prod" {
		if lessonDef.Tier != "prod" {
			log.Errorf("Skipping %s: lower tier than configured", file)
			return fail
		}
	} else if syringeConfig.Tier == "ptr" {
		if lessonDef.Tier != "prod" && lessonDef.Tier != "ptr" {
			log.Errorf("Skipping %s: lower tier than configured", file)
			return fail
		}
	}

	// Ensure there is at least one type of endpoint in the lesson definition
	// TODO(mierdin): If there are only blackboxes, you may also want to validate there are iframes to display them
	if len(lessonDef.Utilities) == 0 && len(lessonDef.Devices) == 0 && len(lessonDef.Blackboxes) == 0 {
		log.Errorf("No endpoints present in %s", file)
		return fail
	}

	// Ensure each device in the definition has a corresponding config for each stage
	for d := range lessonDef.Devices {
		device := lessonDef.Devices[d]
		for s := range lessonDef.Stages {
			stage := lessonDef.Stages[s]
			fileName := fmt.Sprintf("%s/stage%d/configs/%s.txt", filepath.Dir(file), stage.Id, device.Name)
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Configuration for device %s for stage %d was not found.", device.Name, stage.Id)
				return fail
			}
		}
	}

	// Ensure all connections are referring to endpoints that are actually present in the definition
	for c := range lessonDef.Connections {
		connection := lessonDef.Connections[c]

		if !entityInLabDef(connection.A, lessonDef) {
			log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Connection %s refers to nonexistent entity", connection.A)))
			return fail
		}

		if !entityInLabDef(connection.B, lessonDef) {
			log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Connection %s refers to nonexistent entity", connection.B)))
			return fail
		}
	}

	if len(lessonDef.IframeResources) > 0 {
		for i := range lessonDef.IframeResources {
			ifr := lessonDef.IframeResources[i]
			if !entityInLabDef(ifr.Ref, lessonDef) {
				log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Iframe resource refers to nonexistent entity %s", ifr.Ref)))
				return fail
			}
		}
	}

	// TODO(mierdin): Make sure lesson ID, lesson name, stage ID and stage name are unique. If you try to read a value in, make sure it doesn't exist. If it does, error out

	// TODO(mierdin): Need to validate that each name is unique across blackboxes, utilities, and devices.

	// TODO(mierdin): Need to run checks to see that files are located where they need to be. Things like
	// configs, and lesson guides

	// Iterate over stages, and retrieve lesson guide content
	for l := range lessonDef.Stages {
		s := lessonDef.Stages[l]

		if s.VerifyCompleteness == true {
			fileName := fmt.Sprintf("%s/stage%d/verify.py", filepath.Dir(file), s.Id)
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Stage specified VerifyCompleteness but no verify.py script was found: %s", err)
				return fail
			}
		}

		// Validate presence of jupyter notebook in expected location
		if s.JupyterLabGuide == true {
			fileName := fmt.Sprintf("%s/stage%d/notebook.ipynb", filepath.Dir(file), s.Id)
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Stage specified a jupyter notebook lesson guide, but the file was not found: %s", err)
				return fail
			}
		} else {
			fileName := fmt.Sprintf("%s/stage%d/guide.md", filepath.Dir(file), s.Id)
			contents, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Encountered problem reading lesson guide: %s", err)
				return fail
			}
			lessonDef.Stages[l].LabGuide = string(contents)
		}

		if s.VerifyCompleteness == true && s.VerifyObjective == "" {
			log.Error("Must provide a VerifyObjective for stages with VerifyCompleteness set to true")
			return fail
		}
	}

	return nil
}

// entityInLabDef is a helper function to ensure that a device is found by name in a lab definition
func entityInLabDef(entityName string, ld *pb.LessonDef) bool {

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

func foobar() {
	x := struct {
		Foo string
		Bar int
	}{"foo", 2}

	v := reflect.ValueOf(x)

	values := make([]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).Interface()
	}

	fmt.Println(values)
}

func IsEmptyValue(e reflect.Value) bool {
	is_show_checking := false
	var checking_type string
	is_empty := true
	switch e.Type().Kind() {
	case reflect.String:
		checking_type = "string"
		if e.String() != "" {
			// fmt.Println("Empty string")
			is_empty = false
		}
	case reflect.Array:
		checking_type = "array"
		for j := e.Len() - 1; j >= 0; j-- {
			is_empty = IsEmptyValue(e.Index(j))
			if is_empty == false {
				break
			}
			/*if e.Index(j).Float() != 0 {
				// fmt.Println("Empty float")
				is_empty = false
				break
			}*/
		}
	case reflect.Float32, reflect.Float64:
		checking_type = "float"
		if e.Float() != 0 {
			is_empty = false
		}
	case reflect.Int32, reflect.Int64:
		checking_type = "int"
		if e.Int() != 0 {
			is_empty = false

		}
	case reflect.Ptr:
		checking_type = "Ptr"
		if e.Pointer() != 0 {
			is_empty = false
		}
	case reflect.Struct:
		checking_type = "struct"
		for i := e.NumField() - 1; i >= 0; i-- {
			is_empty = IsEmptyValue(e.Field(i))
			// fmt.Println(e.Field(i).Type().Kind())
			if !is_empty {
				break
			}
		}
	default:
		checking_type = "default"
		// is_empty = IsEmptyStruct(e)
	}
	if is_show_checking {
		fmt.Println("Checking type :", checking_type, e.Type().Kind())
	}
	return is_empty
}
