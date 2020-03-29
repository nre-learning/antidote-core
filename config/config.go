package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type AntidoteConfig struct {
	InstanceID string `yaml:"instanceId"`

	Tier          string `yaml:"tier"`
	Domain        string `yaml:"domain"`
	ImageOrg      string `yaml:"imageOrg"`
	GRPCPort      int    `yaml:"grpcPort"`
	HTTPPort      int    `yaml:"httpPort"`
	LiveLessonTTL int    `yaml:"liveLessonTTL"`

	Stats struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"stats"`

	AlwaysPull  bool `yaml:"alwaysPull"`
	AllowEgress bool `yaml:"allowEgress"`

	CertLocation      string `yaml:"certLocation"`
	CurriculumDir     string `yaml:"curriculumDir"`
	CurriculumVersion string `yaml:"curriculumVersion"`

	EnabledServices []string `yaml:"enabledServices"`
}

func LoadConfig(configFile string) (AntidoteConfig, error) {

	// Set a new config with defaults set where relevant
	config := AntidoteConfig{
		Tier:              "prod",
		Domain:            "localhost",
		ImageOrg:          "antidotelabs",
		GRPCPort:          50099,
		HTTPPort:          8086,
		LiveLessonTTL:     30,
		AlwaysPull:        false,
		AllowEgress:       false,
		CertLocation:      "prod/tls-certificate",
		CurriculumVersion: "latest",
		EnabledServices: []string{
			"scheduler",
			"api",
			"stats",
		},
	}

	yamlDef, err := ioutil.ReadFile(configFile)
	if err != nil {
		return AntidoteConfig{}, err
	}
	err = yaml.Unmarshal([]byte(yamlDef), &config)
	if err != nil {
		return AntidoteConfig{}, err
	}

	if config.InstanceID == "" {
		return AntidoteConfig{}, errors.New("InstanceID has no default and must be provided")
	}
	if config.CurriculumDir == "" {
		return AntidoteConfig{}, errors.New("CurriculumDir has no default and must be provided")
	}

	log.Debugf("Antidote config: %s", config.JSON())

	return config, nil

}

func (c *AntidoteConfig) JSON() string {
	configJson, _ := json.Marshal(c)
	return string(configJson)
}

// IsServiceEnabled checks the config for a given service name, and if included,
// returns true. Otherwise, returns false.
func (c *AntidoteConfig) IsServiceEnabled(serviceName string) bool {
	for _, name := range c.EnabledServices {
		if name == serviceName {
			return true
		}
	}
	return false
}
