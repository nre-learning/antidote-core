package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type SyringeConfig struct {
	LessonsDir          string
	Tier                string
	Domain              string
	GRPCPort            int
	HTTPPort            int
	DeviceGCAge         int
	NonDeviceGCAge      int
	HealthCheckInterval int
	TSDBExportInterval  int
	TSDBEnabled         bool
	LessonTTL           int

	LessonsLocal     bool
	LessonRepoRemote string
	LessonRepoBranch string
	LessonRepoDir    string
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

	// +syringeconfig SYRINGE_LESSONS is used to specify the location of the actual "lessons" directory.
	// This directory should contain subdirectories for each lesson underneath it.
	searchDir := os.Getenv("SYRINGE_LESSONS")
	if searchDir == "" {
		return nil, errors.New("SYRINGE_LESSONS is a required variable.")
	} else {
		config.LessonsDir = searchDir
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

	// +syringeconfig SYRINGE_LESSONS_LOCAL is a boolean variable to specify if lessons should
	// be pulled from the local filesystem (true), bypassing the need to clone a repository.
	lessonsLocal, err := strconv.ParseBool(os.Getenv("SYRINGE_LESSONS_LOCAL"))
	if lessonsLocal == false || err != nil {
		config.LessonsLocal = false
	} else {
		config.LessonsLocal = true
	}

	// +syringeconfig SYRINGE_LESSON_REPO_REMOTE is the git repo from which pull lesson content
	remote := os.Getenv("SYRINGE_LESSON_REPO_REMOTE")
	if remote == "" {
		config.LessonRepoRemote = "https://github.com/nre-learning/antidote.git"
	} else {
		config.LessonRepoRemote = remote
	}

	// +syringeconfig SYRINGE_LESSON_REPO_BRANCH is the branch of the git repo where lesson content is located
	branch := os.Getenv("SYRINGE_LESSON_REPO_BRANCH")
	if branch == "" {
		config.LessonRepoBranch = "master"
	} else {
		config.LessonRepoBranch = branch
	}

	// +syringeconfig SYRINGE_LESSON_REPO_DIR specifies where to clone the lesson directory to
	repoDir := os.Getenv("SYRINGE_LESSON_REPO_DIR")
	if repoDir == "" {
		config.LessonRepoDir = "/antidote"
	} else {
		config.LessonRepoDir = repoDir
	}

	// +syringeconfig SYRINGE_LESSON_TTL is the length of time (in minutes) a lesson is allowed
	// to remain active without being interacted with, before it is shut down.
	gc, err := strconv.Atoi(os.Getenv("SYRINGE_LESSON_TTL"))
	if gc == 0 || err != nil {
		config.LessonTTL = 30
	} else {
		config.LessonTTL = gc
	}

	log.Debugf("Syringe config: %s", config.JSON())

	return &config, nil

}
