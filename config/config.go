package config

import (
	"encoding/json"
	"errors"

	log "github.com/sirupsen/logrus"
)

// TODO(mierdin): Consider renaming this package to "services", and placing all services related code in here, including
// interface to define an Antidote service, the pub/sub code, and this config stuff

type AntidoteConfig struct {
	InstanceID string `yaml:"instanceID"`

	Tier          string `yaml:"tier"`
	Domain        string `yaml:"domain"`
	ImageOrg      string `yaml:"imageOrg"`
	GRPCPort      int    `yaml:"grpcPort"`
	HTTPPort      int    `yaml:"httpPort"`
	LiveLessonTTL int    `yaml:"liveLessonTTL"`

	Stats struct {
		Enabled  bool   `yaml:"enabled"`
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"stats"`

	AlwaysPull bool `yaml:"alwaysPull"`
	// PrivilegedImages []string `yaml:"privilegedImages"`
	AllowEgress bool `yaml:"allowEgress"`

	CertLocation      string `yaml:"certLocation"`
	CurriculumDir     string `yaml:"curriculumDir"`
	CurriculumVersion string `yaml:"curriculumVersion"`

	EnabledServices []string `yaml:"enabledServices"`
}

func LoadConfig() (*AntidoteConfig, error) {

	// Set a new config with defaults set where relevant
	config := AntidoteConfig{
		Tier:          "prod",
		Domain:        "localhost",
		ImageOrg:      "antidotelabs",
		GRPCPort:      50099,
		HTTPPort:      8086,
		LiveLessonTTL: 30,
		AlwaysPull:    false,
		// PrivilegedImages: []string{ // TODO(mierdin): Shouldn't be needed now that we have images metadata
		// 	"antidotelabs/vqfx-snap1",
		// 	"antidotelabs/vqfx-snap2",
		// 	"antidotelabs/vqfx-snap3",
		// 	"antidotelabs/cvx",
		// 	"antidotelabs/frr",
		// 	"antidotelabs/pjsua-lindsey",
		// 	"antidotelabs/asterisk",
		// },
		AllowEgress:       false,
		CertLocation:      "prod/tls-cert",
		CurriculumVersion: "latest",
		EnabledServices:   []string{},
	}

	if config.InstanceID == "" {
		return nil, errors.New("InstanceID has no default and must be provided")
	}
	if config.CurriculumDir == "" {
		return nil, errors.New("CurriculumDir has no default and must be provided")
	}

	log.Debugf("Antidote config: %s", config.JSON())

	return &config, nil

}

func (c *AntidoteConfig) JSON() string {
	configJson, _ := json.Marshal(c)
	return string(configJson)
}
