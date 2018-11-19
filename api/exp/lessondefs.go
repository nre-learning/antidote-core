package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func (s *server) ListLessonDefs(ctx context.Context, _ *empty.Empty) (*pb.LessonCategoryMap, error) {

	retMap := map[string]*pb.LessonDefs{}

	// TODO(mierdin): Okay for now, but not super efficient. Should store in category keys when loaded.
	for _, lessonDef := range s.scheduler.LessonDefs {

		// Initialize category
		if _, ok := retMap[lessonDef.Category]; !ok {
			retMap[lessonDef.Category] = &pb.LessonDefs{
				LessonDefs: []*pb.LessonDef{},
			}
		}

		retMap[lessonDef.Category].LessonDefs = append(retMap[lessonDef.Category].LessonDefs, lessonDef)
	}

	return &pb.LessonCategoryMap{
		LessonCategories: retMap,
	}, nil
}

func (s *server) GetLessonDef(ctx context.Context, lid *pb.LessonID) (*pb.LessonDef, error) {

	if _, ok := s.scheduler.LessonDefs[lid.Id]; !ok {
		return nil, errors.New("Invalid lesson ID")
	}

	lessonDef := s.scheduler.LessonDefs[lid.Id]

	log.Debugf("Received request for lesson definition: %v", lessonDef)

	return lessonDef, nil
}

// JSON exports the lesson definition as JSON
// func (ld *pb.LessonDef) JSON() string {
// 	lessonJSON, err := json.MarshalIndent(ld, "", "  ")
// 	if err != nil {
// 		log.Error(err)
// 		return ""
// 	}

// 	return string(lessonJSON)
// }

func ImportLessonDefs(syringeConfig *config.SyringeConfig, lessonDir string) (map[int32]*pb.LessonDef, error) {

	// Get lesson definitions
	fileList := []string{}
	log.Debugf("Searching %s for lesson definitions", lessonDir)
	err := filepath.Walk(lessonDir, func(path string, f os.FileInfo, err error) error {
		syringeFileLocation := fmt.Sprintf("%s/syringe.yaml", path)
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

FILES:
	for f := range fileList {

		file := fileList[f]

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue FILES
		}

		var lessonDef pb.LessonDef
		err = yaml.Unmarshal([]byte(yamlDef), &lessonDef)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
			continue FILES
		}

		// Basic validation from protobuf tags
		err = lessonDef.Validate()
		if err != nil {
			log.Errorf("Basic validation failed on %s: %s", file, err)
			continue FILES
		}

		// More advanced validation
		if syringeConfig.Tier == "prod" {
			if lessonDef.Tier != "prod" {
				log.Errorf("Skipping %s: lower tier than configured", file)
				continue FILES
			}
		} else if syringeConfig.Tier == "ptr" {
			if lessonDef.Tier != "prod" && lessonDef.Tier != "ptr" {
				log.Errorf("Skipping %s: lower tier than configured", file)
				continue FILES
			}
		}

		if len(lessonDef.Utilities) == 0 && len(lessonDef.Devices) == 0 && len(lessonDef.Blackboxes) == 0 {
			log.Errorf("No endpoints present in %s", file)
			continue FILES
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
		}

		// TODO(mierdin): Make sure lesson ID, lesson name, stage ID and stage name are unique. If you try to read a value in, make sure it doesn't exist. If it does, error out

		// TODO(mierdin): Need to run checks to see that files are located where they need to be. Things like
		// configs, and lesson guides

		log.Infof("Successfully imported lesson %d: %s", lessonDef.LessonId, lessonDef.LessonName)
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
