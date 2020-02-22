package db

// ImportCurriculum provides a single function for anything within Syringe to load a Curriculum
// into memory. It goes through the logic of importing and validating everything within a curriculum,
// including lessons, collections, etc. This allows both syrctl and the syringe scheduler to simply
// load things exactly the same way every time.
// func ImportCurriculum(config *config.SyringeConfig) (*pb.Curriculum, error) {

// 	curriculum := &pb.Curriculum{}

// 	collections, err := ImportCollections(config)
// 	if err != nil {
// 		log.Warn(err)
// 	}
// 	curriculum.Collections = collections

// 	lessons, err := ImportLessons(config)
// 	if err != nil {
// 		log.Warn(err)
// 	}
// 	curriculum.Lessons = lessons

// 	return curriculum, nil
// }
