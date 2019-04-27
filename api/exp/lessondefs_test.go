package api

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	"github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// TestValidationMissingConfig
func TestValidationMissingConfig(t *testing.T) {

	lessonDir, err := filepath.Abs("../../hack/mocks/")
	ok(t, err)

	file := fmt.Sprintf("%s/lessons/lesson-missingconfig/syringe.yaml", lessonDir)
	yamlDef, err := ioutil.ReadFile(file)
	ok(t, err)

	var lessonDef pb.LessonDef
	err = yaml.Unmarshal([]byte(yamlDef), &lessonDef)
	if err != nil {
		log.Errorf("Failed to import %s: %s", file, err)
	}
	lessonDef.LessonFile = file

	// Because a config is missing, this should fail to validate
	err = validateLesson(&config.SyringeConfig{
		CurriculumDir: "../antidote",
		Domain:        "localhost",
		Tier:          "prod",
	}, &lessonDef)
	assert(t, err != nil, "")
}
