package config

import (
	"errors"
	"os"
	"strconv"
)

type SyringeConfig struct {
	LessonsDir          string
	Tier                string
	GRPCPort            int
	HTTPPort            int
	DeviceGCAge         int
	NonDeviceGCAge      int
	HealthCheckInterval int
	TSDBExportInterval  int
	TSDBEnabled         bool
	IgnoreDisabled      bool // This ignores the "disabled" field in lesson definitions. Useful for showing lessons in any state when running in dev, etc. Will load lesson regardless.
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
	tier := os.Getenv("SYRINGE_TIER")
	if tier == "" {
		return nil, errors.New("SYRINGE_TIER is a required variable.")
	} else {
		if tier != "prod" && tier != "ptr" && tier != "local" {
			return nil, errors.New("SYRINGE_TIER set to incorrect value")
		} else {
			config.Tier = tier
		}
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

	if os.Getenv("SYRINGE_IGNORE_DISABLED") == "true" {
		config.IgnoreDisabled = true
	} else {
		config.IgnoreDisabled = false
	}

	return &config, nil

}
