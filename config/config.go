package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

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

	InfluxdbEnabled bool
	InfluxURL       string
	InfluxUsername  string
	InfluxPassword  string

	TSDBExportInterval int

	CurriculumLocal      bool
	CurriculumVersion    string
	CurriculumRepoRemote string
	CurriculumRepoBranch string

	AlwaysPull bool

	PrivilegedImages []string

	AllowEgress bool
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

	/*
	   OPTIONAL
	*/

	// +syringeconfig SYRINGE_DOMAIN is used when directing iframe resources to the appropriate place.
	// Specify the full domain you're using to access the environment.
	domain := os.Getenv("SYRINGE_DOMAIN")
	if domain == "" {
		config.Domain = "localhost"
	} else {
		config.Domain = domain
	}

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

	// +syringeconfig SYRINGE_CURRICULUM_VERSION is the version of the curriculum to use.
	version := os.Getenv("SYRINGE_CURRICULUM_VERSION")
	if version == "" {
		// This is used to form docker image refs, so we're specifying "latest" here by default.
		config.CurriculumVersion = "latest"
	} else {
		config.CurriculumVersion = version
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

	// +syringeconfig SYRINGE_INFLUXDB_ENABLED controls whether or not influxdb exports take place.
	// Defaults to false.
	influxdbEnabled, err := strconv.ParseBool(os.Getenv("SYRINGE_INFLUXDB_ENABLED"))
	if influxdbEnabled == false || err != nil {
		config.InfluxdbEnabled = false
	} else {
		config.InfluxdbEnabled = true
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

	// +syringeconfig SYRINGE_ALLOW_EGRESS is a boolean variable to specify if network traffic should be
	// allowed to egress lesson namespaces. Defaults to false. If set to true, no NetworkPolicy will be created
	// for lesson namespaces.
	allowEgress, err := strconv.ParseBool(os.Getenv("SYRINGE_ALLOW_EGRESS"))
	if allowEgress == false || err != nil {
		config.AllowEgress = false
	} else {
		config.AllowEgress = true
	}

	// +syringeconfig SYRINGE_IMAGE_PULL_POLICY is a boolean variable that controls the ImagePullPolicy of all
	// pods within a lesson. Defaults to true, which results in an "Always" ImagePullPolicy. Setting to false
	// will result in an "IfNotPresent" policy.
	val := os.Getenv("SYRINGE_ALWAYS_PULL")
	if alwaysPull, err := strconv.ParseBool(val); err == nil {
		config.AlwaysPull = alwaysPull
	} else {
		config.AlwaysPull = true
	}

	// +syringeconfig SYRINGE_PRIVILEGED_IMAGES is a string slice that specifies which images need privileged
	// access granted to them. This option will eventually be deprecated in favor of a more secure option, but
	// for now, this allows us to at least be selective about what images are granted these privileges - ideally
	// only images which only allow user access from within a VM.
	// Images should be separated by commas, no spaces. Image tags should NOT be included.
	privImages := os.Getenv("SYRINGE_PRIVILEGED_IMAGES")
	if privImages == "" {
		config.PrivilegedImages = []string{
			"antidotelabs/container-vqfx",
			"antidotelabs/vqfx-snap1",
			"antidotelabs/vqfx-snap2",
			"antidotelabs/vqfx-snap3",
			"antidotelabs/vqfx-full",
			"antidotelabs/cvx",
			"antidotelabs/frr",
		}
	} else {
		config.PrivilegedImages = strings.Split(privImages, ",")
	}

	log.Debugf("Syringe config: %s", config.JSON())

	return &config, nil

}
