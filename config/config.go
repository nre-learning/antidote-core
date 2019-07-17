package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type SyringeConfig struct {
	CurriculumDir       string
	Tier                string
	Domain              string
	GRPCPort            int
	HTTPPort            int
	DeviceGCAge         int
	NonDeviceGCAge      int
	HealthCheckInterval int
	LiveLessonTTL       int

	InfluxURL      string
	InfluxUsername string
	InfluxPassword string

	CurriculumLocal      bool
	CurriculumRepoRemote string
	CurriculumRepoBranch string
}

func (c *SyringeConfig) JSON() string {
	configJson, _ := json.Marshal(c)
	return string(configJson)
}

func LoadConfigVars() (*SyringeConfig, error) {

	config := SyringeConfig{}

	/*
		REQUIRED
	*/

	// +syringeconfig SYRINGE_CURRICULUM should point to the directory containing the curriculum for
	// Syringe to serve up. This directory should contain subdirectories like "lessons/", and "collections/"
	curriculumDir := os.Getenv("SYRINGE_CURRICULUM")
	if curriculumDir == "" {
		return nil, errors.New("SYRINGE_CURRICULUM is a required variable.")
	} else {
		config.CurriculumDir = curriculumDir
	}

	// +syringeconfig SYRINGE_DOMAIN is used when directing iframe resources to the appropriate place.
	// Specify the full domain you're using to access the environment.
	domain := os.Getenv("SYRINGE_DOMAIN")
	if domain == "" {
		return nil, errors.New("SYRINGE_DOMAIN is a required variable.")
	} else {
		config.Domain = domain
	}

	/*
		OPTIONAL
	*/

	// +syringeconfig SYRINGE_GRPC_PORT specifies the port on which the GRPC server should listen
	grpcPort, err := strconv.Atoi(os.Getenv("SYRINGE_GRPC_PORT"))
	if grpcPort == 0 || err != nil {
		config.GRPCPort = 50099
	} else {
		config.GRPCPort = grpcPort
	}

	// +syringeconfig SYRINGE_HTTP_PORT specifies the port on which the HTTP/REST API server should listen
	httpPort, err := strconv.Atoi(os.Getenv("SYRINGE_HTTP_PORT"))
	if httpPort == 0 || err != nil {
		config.HTTPPort = 8086
	} else {
		config.HTTPPort = httpPort
	}

	// +syringeconfig SYRINGE_TIER specifies the tier at which Syringe should operate. Valid values include
	// "prod", "ptr", or "local".
	tier := os.Getenv("SYRINGE_TIER")
	if tier == "" {
		config.Tier = "local"
	} else {
		if tier != "prod" && tier != "ptr" && tier != "local" {
			return nil, errors.New("SYRINGE_TIER set to incorrect value")
		} else {
			config.Tier = tier
		}
	}

	// +syringeconfig SYRINGE_CURRICULUM_LOCAL is a boolean variable to specify if the curriculum should
	// be pulled from the local filesystem (true), bypassing the need to clone a repository.
	curriculumLocal, err := strconv.ParseBool(os.Getenv("SYRINGE_CURRICULUM_LOCAL"))
	if curriculumLocal == false || err != nil {
		config.CurriculumLocal = false
	} else {
		config.CurriculumLocal = true
	}

	// +syringeconfig SYRINGE_CURRICULUM_REPO_REMOTE is the git repo from which pull lesson content
	remote := os.Getenv("SYRINGE_CURRICULUM_REPO_REMOTE")
	if remote == "" {
		config.CurriculumRepoRemote = "https://github.com/nre-learning/nrelabs-curriculum.git"
	} else {
		config.CurriculumRepoRemote = remote
	}

	// +syringeconfig SYRINGE_CURRICULUM_REPO_BRANCH is the branch of the git repo where lesson content is located
	branch := os.Getenv("SYRINGE_CURRICULUM_REPO_BRANCH")
	if branch == "" {
		config.CurriculumRepoBranch = "master"
	} else {
		config.CurriculumRepoBranch = branch
	}

	// +syringeconfig SYRINGE_LIVELESSON_TTL is the length of time (in minutes) a lesson is allowed
	// to remain active without being interacted with, before it is shut down.
	gc, err := strconv.Atoi(os.Getenv("SYRINGE_LIVELESSON_TTL"))
	if gc == 0 || err != nil {
		config.LiveLessonTTL = 30
	} else {
		config.LiveLessonTTL = gc
	}

	// +syringeconfig SYRINGE_INFLUXDB_URL is the URL for the influxdb-based metrics server.
	influxURL := os.Getenv("SYRINGE_INFLUXDB_URL")
	if influxURL == "" {
		config.InfluxURL = "https://influxdb.networkreliability.engineering/"
	} else {
		config.InfluxURL = influxURL
	}

	// +syringeconfig SYRINGE_INFLUXDB_USERNAME is the username for the influxdb-based metrics server.
	influxUsername := os.Getenv("SYRINGE_INFLUXDB_USERNAME")
	if influxUsername == "" {
		config.InfluxUsername = "admin"
	} else {
		config.InfluxUsername = influxUsername
	}

	// +syringeconfig SYRINGE_INFLUXDB_PASSWORD is the password for the influxdb-based metrics server.
	influxPassword := os.Getenv("SYRINGE_INFLUXDB_PASSWORD")
	if influxPassword == "" {
		config.InfluxPassword = "zerocool"
	} else {
		config.InfluxPassword = influxPassword
	}

	log.Debugf("Syringe config: %s", config.JSON())

	return &config, nil

}
