package main

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/fatih/color"

	models "github.com/nre-learning/syringe/db/models"
)

func promptForValue(fieldType, fieldPattern, fieldName string) string {
	// Provide type hints and description
	color.White("Please input value for %s", fieldName)
	pattern := ""
	if fieldPattern != "" {
		pattern = fmt.Sprintf(" (Pattern: %s)", fieldPattern)
	}
	fmt.Printf("(Type: %s%s)\n", fieldType, pattern)

	// Read input from user
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	response = strings.Trim(response, "\n")

	// Before returning, should match response
	return response
}

// iterateType
func newLessonWizard() *models.Lesson {
	// jsType jsonschema.Type

	newLesson := models.Lesson{}

	lessonSchema := newLesson.GetSchema()

	lessonType := lessonSchema.Definitions["Lesson"]

	for k, v := range lessonType.Properties {

		// Go through simple properties first
		if v.Type != "array" {

			// TODO I **think** all fields are currently strings right now but we should consider casting here
			userValue := promptForValue(v.Type, v.Pattern, k)

			switch v.Type {
			case "int":
				// reflect.ValueOf(&newLesson).Elem().FieldByName(k).SetInt(userValue)
			default:
				reflect.ValueOf(&newLesson).Elem().FieldByName(k).SetString(userValue)
			}

			// log.Infof("Provided '%s' for field %s", response, k)
		}
	}

	// for k, v := range lessonType.Properties {

	// 	// Next, go through simple arrays that don't have a ref to
	// 	// another type
	// 	if v.Type == "array" && v.Items.Ref == "" {

	// 		// Provide type hints and description
	// 		color.White("Please input value for %s as an array separated by commas", k)
	// 		pattern := ""
	// 		if v.Pattern != "" {
	// 			pattern = fmt.Sprintf(" (Pattern: %s)", v.Pattern)
	// 		}
	// 		fmt.Printf("(Type: %s%s)\n", v.Type, pattern)

	// 		reader := bufio.NewReader(os.Stdin)
	// 		response, err := reader.ReadString('\n')
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		response = strings.Trim(response, "\n")
	// 		log.Infof("Provided '%s' for field %s", response, k)

	// 		switch itemType := v.Items.Type; itemType {
	// 		case "int":
	// 			fmt.Println("OS X.")
	// 		default:
	// 			// Handle as a string
	// 			fmt.Printf("%s.\n", itemType)
	// 		}
	// 	}
	// }

	// if "items" is not nil, and "schema" of "items" is not nil, then you have a subtype
	// If "items" is not nil but there is no "schema" key off of that, then it's an array

	return &newLesson

}
