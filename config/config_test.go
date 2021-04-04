package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/nats-io/nats.go"
)

// Helper functions courtesy of the venerable Ben Johnson
// https://medium.com/@benbjohnson/structuring-tests-in-go-46ddee7a25c

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// TestConfigJSON ensures a given config renders correctly as JSON
func TestConfigJSON(t *testing.T) {
	antidoteConfig, err := LoadConfig("../hack/mocks/sample-antidote-config.yml")
	if err != nil {
		t.Fatal(err)
	}

	desired := AntidoteConfig{
		CurriculumDir:             "/usr/bin/curriculum",
		InstanceID:                "foobar",
		Tier:                      "prod",
		ImageOrg:                  "antidotelabs",
		HEPSDomain:                "heps.nrelabs.io",
		GRPCPort:                  50099,
		HTTPPort:                  8086,
		LiveSessionTTL:            1440,
		LiveLessonTTL:             30,
		LiveSessionLimit:          0,
		LiveLessonLimit:           0,
		CurriculumVersion:         "latest",
		AlwaysPull:                false,
		SecretsNamespace:          "prod",
		TLSCertName:               "tls-certificate",
		PullCredName:              "",
		AllowEgress:               false,
		EnabledServices:           []string{"foobarsvc"},
		K8sInCluster:              true,
		K8sOutOfClusterConfigPath: "",
		NATSUrl:                   nats.DefaultURL,
		DevMode:                   false,
	}

	t.Log(antidoteConfig.JSON())
	t.Log(desired.JSON())

	// Pretty barbaric but works for now
	assert(t, antidoteConfig.JSON() == desired.JSON(), "")
}
