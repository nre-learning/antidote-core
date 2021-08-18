package db

import (
	"testing"

	"github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
)

// getValidImage returns a full, valid example of an Image that uses all the features.
// Tests in this file should make use of this by making a copy, tweaking in some way that makes it
// invalid, and then asserting on the error type/message.
func getValidImage() models.Image {
	images, err := ReadImages(config.AntidoteConfig{
		CurriculumDir: "../test/test-curriculum",
		Tier:          "local",
	})
	if err != nil {
		panic(err)
	}

	for _, i := range images {
		if i.Slug == "utility" {
			return *i
		}
	}
	panic("unable to find valid image")
}

func TestValidImage(t *testing.T) {
	i := getValidImage()
	err := validateImage(&i)
	assert(t, (err == nil), "Expected validation to pass, but encountered validation errors")
}

func TestInvalidCharInImageSlug(t *testing.T) {
	i := getValidImage()
	i.Slug = "antidotelabs/utility:latest"
	err := validateImage(&i)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}

func TestNoNetworkInterfaces(t *testing.T) {
	i := getValidImage()
	i.NetworkInterfaces = []string{}
	err := validateImage(&i)

	assert(t, (err == nil), "Expected no error; the NetworkInterfaces field is optional")
}

func TestInvalidNetworkInterface(t *testing.T) {
	i := getValidImage()
	i.NetworkInterfaces = []string{"eth0", "net1"}
	err := validateImage(&i)

	assert(t, (err == errEth0NotAllowed), "Expected errEth0NotAllowed")
}

func TestNoSSHUser(t *testing.T) {
	i := getValidImage()
	i.SSHUser = ""
	err := validateImage(&i)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}
