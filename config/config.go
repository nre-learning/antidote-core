package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type AntidoteConfig struct {
	InstanceID string `yaml:"instanceId"`

	Tier string `yaml:"tier"`

	// HepsDomain is the domain on which HEPS hostnames should be provisioned. This domain is used
	// as the direct parent for any subdomains that are configured for ingresses that allow access to
	// HTTP endpoints. Note that these subdomains automatically include the antidote instance ID, so
	// it's safe to have multiple instances of antidote running with the same HepsDomain
	// configured.
	HEPSDomain string `yaml:"hepsDomain"`

	ImageOrg string `yaml:"imageOrg"`
	GRPCPort int    `yaml:"grpcPort"`
	HTTPPort int    `yaml:"httpPort"`

	// Both TTL options are in minutes
	LiveSessionTTL int `yaml:"liveSessionTTL"`
	LiveLessonTTL  int `yaml:"liveLessonTTL"`

	// Limits the number of sessions that can be requested by a single IP address
	LiveSessionLimit int `yaml:"liveSessionLimit"`

	// Less important but still useful. LiveLessons are already bound to a session ID, so the
	// number of livelessons that can be spun up by a single session is bound to the number of
	// lessons in a given curriculum. This option, however, can be used to further limit the number
	// of concurrent livelessons for a single session
	LiveLessonLimit int `yaml:"liveLessonLimit"`

	Stats struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"stats"`

	AlwaysPull bool `yaml:"alwaysPull"`

	// ALL backends must default to a secure network policy, by restricting traffic outside the smallest unit necessary to
	// make intra-lesson traffic possible. This config option can be used to instruct the backend to change this policy to allow traffic outside
	// this boundary. For the kubernetes backend, this boundary is the namespace.
	AllowEgress bool `yaml:"allowEgress"`

	CurriculumDir     string `yaml:"curriculumDir"`
	CurriculumVersion string `yaml:"curriculumVersion"`

	EnabledServices []string `yaml:"enabledServices"`

	NATSUrl string `yaml:"natsUrl"`

	// ONLY meant to be used for development purposes. Statically sets session ID, and a few other things useful
	// for development purposes.
	DevMode bool `yaml:"devMode"`

	Backend        string         `yaml:"backend"`
	BackendConfigs BackendConfigs `yaml:"backendConfigs"`
}

// BackendConfigs holds configuration options that are specific to a particular backend. The field in BackendConfigs
// should match the backend name it refers to, for consistency.
type BackendConfigs struct {
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
}

type KubernetesConfig struct {
	// InCluster controls whether or not the kubernetes backend uses an in-cluster
	// configuration for communicating with kubernetes. Since this is the typical deployment
	// scenario, this defaults to true.
	//
	// However, for development, it may be useful to use an out of cluster configuration,
	// so you can run antidoted directly instead of packaging it in a container image and deploying
	// to your cluster. In this case, set InCluster to false, and provide the path to
	// your kubeconfig via OutOfClusterConfigPath
	InCluster              bool   `yaml:"inCluster"`
	OutOfClusterConfigPath string `yaml:"outOfClusterConfigPath"`

	SecretsNamespace string `yaml:"secretsNamespace"`
	TLSCertName      string `yaml:"tlsCertName"`
	// https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	PullCredName string `yaml:"pullCredName"`
}

func LoadConfig(configFile string) (AntidoteConfig, error) {

	// Set a new config with defaults set where relevant
	config := AntidoteConfig{
		Tier:              "prod",
		HEPSDomain:        "localhost",
		ImageOrg:          "antidotelabs",
		GRPCPort:          50099,
		HTTPPort:          8086,
		LiveSessionTTL:    1440,
		LiveLessonTTL:     30,
		LiveSessionLimit:  0,
		LiveLessonLimit:   0,
		AlwaysPull:        false,
		AllowEgress:       false,
		CurriculumVersion: "latest",
		EnabledServices: []string{
			"scheduler",
			"api",
			"stats",
		},
		NATSUrl: nats.DefaultURL,
		DevMode: false,
		BackendConfigs: BackendConfigs{
			Kubernetes: KubernetesConfig{
				InCluster:              true,
				OutOfClusterConfigPath: "",
				SecretsNamespace:       "prod",
				TLSCertName:            "tls-certificate",
				PullCredName:           "",
			},
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
