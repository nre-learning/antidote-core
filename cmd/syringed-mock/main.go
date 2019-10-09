package main

import (
	pb "github.com/nre-learning/syringe/api/exp/generated"
	config "github.com/nre-learning/syringe/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {

	syringeConfig := &config.SyringeConfig{
		Domain:   "localhost",
		GRPCPort: 50099,
		HTTPPort: 8086,
	}

	var lesson *pb.Lesson
	err := yaml.Unmarshal([]byte(lessonRaw), &lesson)
	if err != nil {
		log.Fatal(err)
	}

	var collection *pb.Collection
	err = yaml.Unmarshal([]byte(collectionRaw), &collection)
	if err != nil {
		log.Fatal(err)
	}

	apiServer := &MockAPIServer{
		Lessons: []*pb.Lesson{
			lesson,
		},
		Collections: []*pb.Collection{
			collection,
		},
	}
	err = apiServer.StartAPI(syringeConfig)
	if err != nil {
		log.Fatalf("Problem starting API: %s", err)
	}

	ch := make(chan struct{})
	<-ch
}
