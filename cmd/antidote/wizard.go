package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey"
	jsonschema "github.com/alecthomas/jsonschema"
	"github.com/fatih/color"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func promptForValue(name string, value *jsonschema.Type) string {
	var q = &survey.Question{
		// This function interacts with the user by asking single-question surveys for each field. So, for
		// predictable outcomes when using the survey package, we'll statically set the "Name" field to
		// the string "name", and then retrieve that single field from the resulting struct before returning
		// the value to the caller
		Name: "value",
		Validate: func(val interface{}) error {
			return nil
		},
	}

	pattern := ""
	if value.Pattern != "" {
		pattern = fmt.Sprintf(", pattern: %s", value.Pattern)
	}
	valueType := value.Type
	if valueType == "array" {
		valueType = "comma-separated array"
	}

	reqd := "optional"
	if value.MinLength > 0 {
		reqd = "required"
	}

	help := fmt.Sprintf("%s (%s,%s%s) %s", name, reqd, valueType, pattern, value.Description)

	if len(value.Enum) > 0 {
		opts := []string{}
		for _, opt := range value.Enum {
			if opt == "" {
				opt = "<none>"
			}
			opts = append(opts, opt.(string))
		}
		q.Prompt = &survey.Select{
			Message: fmt.Sprintf("%s:", name),
			Help:    help,
			Options: opts,
		}
	} else {
		q.Prompt = &survey.Input{
			Message: fmt.Sprintf("%s:", name),
			Help:    help,
		}
	}

	answers := struct {
		Value string
	}{}
	err := survey.Ask([]*survey.Question{q}, &answers)
	if err == terminal.InterruptErr {
		fmt.Println("Exiting.")
		os.Exit(0)
	} else if err != nil {
		// panic(err)
	}

	if answers.Value == "<none>" {
		return ""
	}

	return answers.Value
}

// schemaWizard takes a jsonschema object, and returns a fully populated map ready to be exported to JSON (and then
// presumably unmarshaled into a type)
//
// rootType specifies the type within this json schema object that we should iterate from (the highest supertype)
// typePrefix is meant to provide nested types with a prefix (only used when this function is called recursively)
func schemaWizard(schema *jsonschema.Schema, root, typePrefix string) (map[string]interface{}, error) {

	rootType := schema.Definitions[root]
	retMap := make(map[string]interface{})

	for i := range rootType.Properties.Keys() {

		k := rootType.Properties.Keys()[i]
		vpre, _ := rootType.Properties.Get(k)
		typeName := fmt.Sprintf("%s%s", typePrefix, k)
		v := vpre.(*jsonschema.Type)

		if v.Type != "array" {
			retString := promptForValue(typeName, v)
			if v.Type == "integer" {
				i, err := strconv.Atoi(retString)
				if err != nil {
					fmt.Printf("Warning - skipping non-integer '%s'", retString)
				}
				retMap[k] = i
			} else {
				retMap[k] = retString
			}
		}

		if v.Type == "array" && v.Items.Ref == "" {
			retString := promptForValue(typeName, v)
			if retString != "" {
				if v.Items.Type == "integer" {
					intArray := []int{}
					for _, member := range strings.Split(retString, ",") {
						i, err := strconv.Atoi(member)
						if err != nil {
							fmt.Printf("Warning - dropping non-integer '%s'", member)
							continue
						}
						intArray = append(intArray, i)
					}
					retMap[k] = intArray
				} else {
					retMap[k] = strings.Split(retString, ",")
				}
			}
		}

		// Complex type. Recurse into this function with the new root type
		if v.Type == "array" && v.Items.Ref != "" {

			splitSlice := strings.Split(v.Items.Ref, "/")
			subTypeName := splitSlice[len(splitSlice)-1]

			if v.MinItems == 0 {
				if !simpleConfirm(fmt.Sprintf("--- Do you wish to create any %s? ---", typeName)) {
					continue
				}
			}
			color.Yellow("You will now be prompted to create a series of %s (%s)\n", typeName, v.Description)

			var members []interface{}

			i := 0
			for {
				innerMap, _ := schemaWizard(schema, subTypeName, fmt.Sprintf("%s[%d].", typeName, i))
				members = append(members, innerMap)

				if !simpleConfirm(fmt.Sprintf("--- Do you want to add more %s? ---", typeName)) {
					break
				}

				i++
			}
			retMap[k] = members
		}

	}

	return retMap, nil

}
