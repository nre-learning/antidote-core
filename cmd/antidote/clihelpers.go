package main

import (
	"fmt"

	"github.com/AlecAivazis/survey"
)

func askSimpleValue(prompt, defaultValue string) string {
	var val survey.Validator
	resp := ""
	q := &survey.Input{
		Message: fmt.Sprintf("%s [%s]:", prompt, defaultValue),
	}
	survey.AskOne(q, &resp, val)
	return resp
}

func simpleConfirm(msg string) bool {
	var val survey.Validator
	resp := false
	prompt := &survey.Confirm{
		Message: msg,
	}
	survey.AskOne(prompt, &resp, val)
	return resp
}
