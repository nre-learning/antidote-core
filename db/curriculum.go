package db

import config "github.com/nre-learning/syringe/config"

// TODO(mierdin): Consider just adding these as records to meta

type Curriculum struct {
	Name                  string
	Description           string
	Website               string
	TargetAntidoteVersion string
}

// TODO(mierdin): Curriculum will no longer be a first-class type, but rather will be imported as simple k/v pairs
// into the Meta table
func ImportCurriculum(config *config.SyringeConfig) error {
	// curriculum := &models.Curriculum{}
	return nil
}
