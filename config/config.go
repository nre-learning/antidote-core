package config

import (
	"errors"
	"os"
	"strconv"
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
	GCInterval          int

	LessonRepoRemote string
	LessonRepoBranch string
	LessonRepoDir    string
}

func LoadConfigVars() (*SyringeConfig, error) {

	config := SyringeConfig{}

	/*
		REQUIRED
	*/

	// Get configuration parameters from env
	searchDir := os.Getenv("SYRINGE_LESSONS")
	if searchDir == "" {
		return nil, errors.New("SYRINGE_LESSONS is a required variable.")
	} else {
		config.LessonsDir = searchDir
	}

	domain := os.Getenv("SYRINGE_DOMAIN")
	if domain == "" {
		return nil, errors.New("SYRINGE_DOMAIN is a required variable.")
	} else {
		config.Domain = domain
	}

	/*
		OPTIONAL
	*/
	grpcPort, err := strconv.Atoi(os.Getenv("SYRINGE_GRPC_PORT"))
	if grpcPort == 0 || err != nil {
		config.GRPCPort = 50099
	} else {
		config.GRPCPort = grpcPort
	}
	httpPort, err := strconv.Atoi(os.Getenv("SYRINGE_HTTP_PORT"))
	if httpPort == 0 || err != nil {
		config.HTTPPort = 8086
	} else {
		config.HTTPPort = httpPort
	}

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

	remote := os.Getenv("SYRINGE_LESSON_REPO_REMOTE")
	if remote == "" {
		config.LessonRepoRemote = "https://github.com/nre-learning/antidote.git"
	} else {
		config.LessonRepoRemote = remote
	}

	branch := os.Getenv("SYRINGE_LESSON_REPO_BRANCH")
	if branch == "" {
		config.LessonRepoBranch = "master"
	} else {
		config.LessonRepoBranch = branch
	}

	dir := os.Getenv("SYRINGE_LESSON_REPO_DIR")
	if dir == "" {
		config.LessonRepoDir = "/antidote"
	} else {
		config.LessonRepoDir = dir
	}

	gc, err := strconv.Atoi(os.Getenv("SYRINGE_GC_INTERVAL"))
	if gc == 0 || err != nil {
		config.GCInterval = 5
	} else {
		config.GCInterval = gc
	}

	return &config, nil

}
