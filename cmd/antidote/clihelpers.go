package main

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func askSimpleValue(prompt, defaultValue, help string) string {
	var val survey.Validator
	resp := ""
	q := &survey.Input{
		Message: fmt.Sprintf("%s:", prompt),
		Default: defaultValue,
		Help:    help,
	}
	err := survey.AskOne(q, &resp, val)
	if err == terminal.InterruptErr {
		fmt.Println("Exiting.")
		os.Exit(0)
	} else if err != nil {
		// panic(err)
	}
	return resp
}

func simpleConfirm(msg, help string) bool {
	var val survey.Validator
	resp := false
	prompt := &survey.Confirm{
		Message: msg,
		Help:    help,
	}
	err := survey.AskOne(prompt, &resp, val)
	if err == terminal.InterruptErr {
		fmt.Println("Exiting.")
		os.Exit(0)
	} else if err != nil {
		// panic(err)
	}

	return resp
}
