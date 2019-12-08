package main

import (
	"path/filepath"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"github.com/golang/protobuf/ptypes/timestamp"
	"time"
	config "github.com/nre-learning/syringe/config"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	yaml "gopkg.in/yaml.v2"
	log "github.com/sirupsen/logrus"
)
// mock functions on importing lessons from api/exp/curricula & api/exp/lessons
func GetCurriculum (mockConfig *config.SyringeConfig) *pb.Curriculum {
	curriculum := &pb.Curriculum{}

	lessons, err := ImportLessons(mockConfig)
	if err != nil {
		log.Warn(err)
	}
	curriculum.Lessons = lessons

	return curriculum
}

func GetMockLiveLessonState() map[string]*pb.LiveLesson {
	lesson := pb.LiveLesson{}

	lesson.LessonUUID = "14-4kfl6n3terlzxa3s"
	lesson.LessonId = 14

	presentations := pb.Presentation{}
	presentations.Name = "cli"
	presentations.Port = 22
	presentations.Type = "ssh"

	endpoint := pb.Endpoint{}
	endpoint.Name = "linux1"
	endpoint.Host = "0.0.0.0"
	endpoint.Presentations = []*pb.Presentation{&presentations}

	lesson.LiveEndpoints = map[string]*pb.Endpoint {
		"linux1": &endpoint,
	}

	lesson.LabGuide = "# Introduction to YAML\n## Part 1 - Lists\n\nWelcome to this introduction to YAML! From the very first moment you started looking into network automation, chances are you keep\nhearing about YAML. The reason for this is that YAML is a simple way to describe common data structures in a format that's both\neasily understood by humans, as well as easily parseable by machines. As a result, it powers a large number of automation tools,\nboth inside and outside of networking.\n\nTo prepare you for more advanced lessons that might use YAML, we want to spend some time covering the basics, so you're able to\nlook at an existing YAML document and understand it, or even create your own. This will allow you to do things like write Ansible playbooks,\nJSNAPy tests, and much more.\n\nAs mentioned, YAML lets us represent simple data structures in a text format. One such data structure is the `list`. Most of YAML's capabilities\nclosely mirror Python's data structures, and the `list` is a prime example. Let's take a look at a sample YAML list:\n\n```\ncd /antidote/stage1/\ncat list.yaml\n```\n<button type=\"button\" class=\"btn btn-primary btn-sm\" onclick=\"runSnippetInTab('linux1', this)\">Run this snippet</button>\n\nIn this lesson, we'll work with this YAML data using the interactive Python shell. Run the below snippet to load up our YAML file in Python:\n\n```\npython\nimport yaml\nimport sys\nyamlFile = open('list.yaml', 'r')\nyamlList = yaml.load(yamlFile)\n```\n<button type=\"button\" class=\"btn btn-primary btn-sm\" onclick=\"runSnippetInTab('linux1', this)\">Run this snippet</button>\n\nAt this point, `yamlList` is a Python list that contains values (in this case, strings) that were in our YAML file. We can start by checking this list's length:\n\n```\nprint(\"There are %d values in this YAML file\" % len(yamlList))\n```\n<button type=\"button\" class=\"btn btn-primary btn-sm\" onclick=\"runSnippetInTab('linux1', this)\">Run this snippet</button>\n"
	lesson.LiveLessonStatus = pb.Status_READY

	createdTime, _ := time.Parse(time.RFC3339, "2019-12-08T01:34:58Z")
	lesson.CreatedTime = &timestamp.Timestamp{
		Seconds: createdTime.Unix(),
	}

	lesson.LessonStage = 1
	lesson.LessonStage = 1
	lesson.HealthyTests = 1
	lesson.TotalTests = 1

	return map[string]*pb.LiveLesson{
		"14-4kfl6n3terlzxa3s": &lesson,
	}
}

func GetmockSyringeConfig() *config.SyringeConfig {
	mockSyringeConfig := config.SyringeConfig{}
	mockSyringeConfig.CurriculumDir = "/antidote"
	mockSyringeConfig.Tier = "local"
	mockSyringeConfig.Domain = "antidote-local"
	mockSyringeConfig.GRPCPort = 50099
	mockSyringeConfig.HTTPPort = 8086
	mockSyringeConfig.DeviceGCAge = 0
	mockSyringeConfig.NonDeviceGCAge = 0
	mockSyringeConfig.HealthCheckInterval = 0
	mockSyringeConfig.LiveLessonTTL = 30
	mockSyringeConfig.InfluxdbEnabled = true
	mockSyringeConfig.InfluxURL = "http://localhost:8087"
	mockSyringeConfig.InfluxUsername = "syringe_username"
	mockSyringeConfig.InfluxPassword = "syringe_password"
	mockSyringeConfig.TSDBExportInterval = 0
	mockSyringeConfig.CurriculumLocal = true
	mockSyringeConfig.CurriculumVersion = "latest"
	mockSyringeConfig.CurriculumRepoRemote = "https://github.com/nre-learning/nrelabs-curriculum.git"
	mockSyringeConfig.CurriculumRepoBranch = "master"
	mockSyringeConfig.AllowEgress = false

	privilegedImages := []string{
		"antidotelabs/container-vqfx",
		"antidotelabs/vqfx-snap1",
		"antidotelabs/vqfx-snap2",
		"antidotelabs/vqfx-snap3",
		"antidotelabs/vqfx-full",
		"antidotelabs/cvx",
		"antidotelabs/frr",
	}

	mockSyringeConfig.PrivilegedImages = privilegedImages

	return &mockSyringeConfig
}

func ImportLessons(syringeConfig *config.SyringeConfig) (map[int32]*pb.Lesson, error) {

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

	retLds := map[int32]*pb.Lesson{}

	for f := range fileList {

		file := fileList[f]

		log.Infof("Importing lesson definition at: %s", file)

		yamlDef, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Encountered problem %s", err)
			continue
		}

		var lesson pb.Lesson
		err = yaml.Unmarshal([]byte(yamlDef), &lesson)
		if err != nil {
			log.Errorf("Failed to import %s: %s", file, err)
		}
		lesson.LessonFile = file
		lesson.LessonDir = filepath.Dir(file)

		if _, ok := retLds[lesson.LessonId]; ok {
			log.Errorf("Failed to import %s: Lesson ID %d already exists in another lesson definition.", file, lesson.LessonId)
			continue
		}

		err = validateLesson(syringeConfig, &lesson)
		if err != nil {
			continue
		}
		log.Infof("Successfully imported lesson %d: %s  with %d endpoints.", lesson.LessonId, lesson.LessonName, len(lesson.Endpoints))

		// Insert stage at zero-index so we can use slice indexes to refer to each stage without jumping through hoops
		// or making the user use 0 as a stage ID
		lesson.Stages = append([]*pb.LessonStage{{Id: 0}}, lesson.Stages...)

		retLds[lesson.LessonId] = &lesson
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
func validateLesson(syringeConfig *config.SyringeConfig, lesson *pb.Lesson) error {

	// TODO(mierdin): In the future, you should consider putting unique error messages for
	// each violation. This will make this function more testable.
	fail := errors.New("failed to validate lesson definition")

	file := lesson.LessonFile

	// Basic validation from protobuf tags
	err := lesson.Validate()
	if err != nil {
		log.Errorf("Basic validation failed on %s: %s", file, err)
		return fail
	}

	// More advanced validation
	if syringeConfig.Tier == "prod" {
		if lesson.Tier != "prod" {
			log.Errorf("Skipping %s: lower tier than configured", file)
			return fail
		}
	} else if syringeConfig.Tier == "ptr" {
		if lesson.Tier != "prod" && lesson.Tier != "ptr" {
			log.Errorf("Skipping %s: lower tier than configured", file)
			return fail
		}
	}

	// Ensure each device in the definition has a corresponding config for each stage
	for i := range lesson.Endpoints {

		ep := lesson.Endpoints[i]

		if len(ep.Presentations) == 0 && len(ep.AdditionalPorts) == 0 {
			log.Error("No presentations configured, and no additionalPorts specified")
			return fail
		}

		if strings.Contains(ep.Image, ":") {
			log.Error("Tags are not allowed in endpoint image refs")
			return fail
		}

		if ep.ConfigurationType == "" {
			continue
		}

		supportedConfigurationOptions := []string{
			"python",
			// "bash",  // not yet
			"ansible",
			"napalm-.*$",
		}

		matchedOne := false
		for o := range supportedConfigurationOptions {
			matched, err := regexp.Match(supportedConfigurationOptions[o], []byte(ep.ConfigurationType))
			if err != nil {
				log.Error("Unable to determine configurationType")
				return fail
			}
			if matched {
				matchedOne = true
				break
			}
		}
		if !matchedOne {
			log.Error("Unsupported configurationType")
			return fail
		}

		fileMap := map[string]string{
			"python": ".py",
			// "bash":    ".sh",  // not yet
			"ansible": ".yml",
			"napalm":  ".txt",
		}

		// all napalm configs need to have the same file extension so we're just simplifying for this import
		// validation.
		simpleConfigType := ep.GetConfigurationType()
		if strings.Contains(simpleConfigType, "napalm") {
			simpleConfigType = "napalm"
		}

		// Ensure the necessary config file is present for all stages
		for s := range lesson.Stages {
			fileName := fmt.Sprintf("%s/stage%d/configs/%s%s", filepath.Dir(file), lesson.Stages[s].Id, ep.Name, fileMap[simpleConfigType])
			_, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Errorf("Configuration script %s was not found.", fileName)
				return fail
			}
		}

		// Ensure each presentation name is unique for each endpoint
		seenPresentations := map[string]*pb.Presentation{}
		for n := range ep.Presentations {
			if _, ok := seenPresentations[ep.Presentations[n].Name]; ok {
				log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Presentation %s appears more than once for an endpoint", ep.Presentations[n].Name)))
				return fail
			}

			if ep.Presentations[n].Port == 0 {
				log.Error("All presentations must specify a port")
				return fail
			}

			seenPresentations[ep.Presentations[n].Name] = ep.Presentations[n]
		}
	}

	// Ensure all connections are referring to endpoints that are actually present in the definition
	for c := range lesson.Connections {
		connection := lesson.Connections[c]

		if !entityInLabDef(connection.A, lesson) {
			log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Connection %s refers to nonexistent entity", connection.A)))
			return fail
		}

		if !entityInLabDef(connection.B, lesson) {
			log.Errorf("Failed to import %s: %s", file, errors.New(fmt.Sprintf("Connection %s refers to nonexistent entity", connection.B)))
			return fail
		}
	}

	// TODO(mierdin): Make sure lesson ID, lesson name, stage ID and stage name are unique. If you try to read a value in, make sure it doesn't exist. If it does, error out

	// TODO(mierdin): Need to validate that each name is unique across endpoints

	// TODO(mierdin): Need to run checks to see that files are located where they need to be. Things like
	// configs, and lesson guides

	// Iterate over stages, and retrieve lesson guide content
	for l := range lesson.Stages {
		s := lesson.Stages[l]

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
			lesson.Stages[l].LabGuide = string(contents)
		}

		if s.VerifyCompleteness == true && s.VerifyObjective == "" {
			log.Error("Must provide a VerifyObjective for stages with VerifyCompleteness set to true")
			return fail
		}

		//TODO(mierdin): How to check to make sure referenced collection exists
	}

	return nil
}

// entityInLabDef is a helper function to ensure that a device is found by name in a lab definition
func entityInLabDef(entityName string, ld *pb.Lesson) bool {

	for i := range ld.Endpoints {
		if entityName == ld.Endpoints[i].Name {
			return true
		}
	}
	return false
}
