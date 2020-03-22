package main

import (
	"fmt"

	"github.com/AlecAivazis/survey"

	log "github.com/sirupsen/logrus"
)

// Credit to https://gist.github.com/albrow/5882501
// askForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

func askSimpleValue(prompt, defaultValue string) string {
	var val survey.Validator
	resp := ""
	q := &survey.Input{
		Message: fmt.Sprintf("%s [%s]:", prompt, defaultValue),
	}
	survey.AskOne(q, &resp, val)
	return resp
}

// Uses the survey.Confirm prompt to gather a simple yes/no response
func simpleConfirm(msg string) bool {
	var val survey.Validator
	resp := false
	prompt := &survey.Confirm{
		Message: msg,
	}
	survey.AskOne(prompt, &resp, val)
	return resp
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}
