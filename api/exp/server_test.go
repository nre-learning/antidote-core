package api

import (
	config "github.com/nre-learning/antidote-core/config"
	db "github.com/nre-learning/antidote-core/db"
	ingestors "github.com/nre-learning/antidote-core/db/ingestors"
)

func createFakeAPIServer() *AntidoteAPI {
	cfg, err := config.LoadConfig("../../hack/mocks/mock-config-2.yml")
	if err != nil {
		panic(err)
	}

	// Initialize DataManager
	adb := db.NewADMInMem()
	err = ingestors.ImportCurriculum(adb, cfg)
	if err != nil {
		panic(err)
	}

	// Start API server
	lessonAPIServer := AntidoteAPI{
		Config: cfg,
		Db:     adb,
	}

	return &lessonAPIServer
}
