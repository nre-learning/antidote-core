package api

import (
	"context"
	"errors"

	copier "github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"

	pb "github.com/nre-learning/syringe/api/exp/generated"
	models "github.com/nre-learning/syringe/db/models"
)

// ListCollections returns a list of Collections present in the data store
func (s *AntidoteAPI) ListCollections(ctx context.Context, _ *pb.CollectionFilter) (*pb.Collections, error) {

	collections := []*pb.Collection{}

	dbCollections, err := s.Db.ListCollections()
	if err != nil {
		log.Error(err)
		return nil, errors.New("Error retrieving specified collection")
	}

	for _, c := range dbCollections {
		collections = append(collections, collectionDBToAPI(&c))
	}

	return &pb.Collections{
		Collections: collections,
	}, nil
}

// GetCollection retrieves a single Collection from the data store by Slug
func (s *AntidoteAPI) GetCollection(ctx context.Context, collectionSlug *pb.CollectionSlug) (*pb.Collection, error) {

	dbCollection, err := s.Db.GetCollection(collectionSlug.Slug)
	if err != nil {
		log.Error(err)
		return nil, errors.New("Error retrieving specified collection")
	}

	collection := collectionDBToAPI(dbCollection)

	lessons, err := s.Db.ListLessons()
	if err != nil {
		log.Errorf("Error retrieving lessons for collection %s: %v", dbCollection.Slug, err)
		return nil, errors.New("Error retrieving specified collection")
	}

	for lessonSlug, lesson := range lessons {
		if lesson.Collection == collectionSlug.Slug {
			collection.Lessons = append(collection.Lessons, &pb.LessonSummary{
				LessonSlug:        lessonSlug,
				LessonDescription: lesson.Description,
				LessonName:        lesson.Name,
			})
		}
	}

	return collection, nil
}

// collectionDBToAPI translates a single Collection from the `db` package models into the
// api package's equivalent
func collectionDBToAPI(dbCollection *models.Collection) *pb.Collection {
	collectionAPI := &pb.Collection{}
	copier.Copy(&collectionAPI, dbCollection)
	return collectionAPI
}

// collectionAPIToDB translates a single Collection from the `api` package models into the
// `db` package's equivalent
func collectionAPIToDB(pbCollection *pb.Collection) *models.Collection {
	collectionDB := &models.Collection{}
	copier.Copy(&pbCollection, collectionDB)
	return collectionDB
}
