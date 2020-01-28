package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	jsonschema "github.com/alecthomas/jsonschema"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

func promptForValue(name string, value *jsonschema.Type) string {
	pattern := ""
	if value.Pattern != "" {
		pattern = fmt.Sprintf(", pattern: %s", value.Pattern)
	}
	valueType := value.Type
	if valueType == "array" {
		valueType = "comma-separated array"
	}
	fmt.Printf("%s (%s%s): ", name, value.Type, pattern)

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

func sortProperties(properties map[string]*jsonschema.Type) []string {

	var retProperties []string
	var simpleTypes []string
	var arrayTypes []string
	var complexTypes []string

	for k, v := range properties {
		if v.Type != "array" {
			simpleTypes = append(simpleTypes, k)
		} else if v.Type == "array" && v.Items.Ref == "" {
			arrayTypes = append(arrayTypes, k)
		} else if v.Type == "array" && v.Items.Ref != "" {
			complexTypes = append(complexTypes, k)
		}
	}

	sort.Strings(simpleTypes)
	sort.Strings(arrayTypes)
	sort.Strings(complexTypes)

	retProperties = append(retProperties, simpleTypes...)
	retProperties = append(retProperties, arrayTypes...)
	retProperties = append(retProperties, complexTypes...)

	return retProperties
}

// schemaWizard takes a jsonschema object, and returns a fully populated map ready to be exported to JSON (and then
// presumably unmarshaled into a type)
//
// rootType specifies the type within this json schema object that we should iterate from (the highest supertype)
// typePrefix is meant to provide nested types with a prefix (only used when this function is called recursively)
func schemaWizard(schema *jsonschema.Schema, root, typePrefix string) (map[string]interface{}, error) {

	rootType := schema.Definitions[root]
	retMap := make(map[string]interface{})

	// TODO(mierdin): Need to figure out a way to sort these so that the simple ones are always first, and then
	// within that, sort alphabetically.

	props := sortProperties(rootType.Properties)
	for i := range props {
		// for k, v := range rootType.Properties {

		k := props[i]
		v := rootType.Properties[k]

		typeName := fmt.Sprintf("%s%s", typePrefix, k)

		// Simple type. Just prompt for value.
		if v.Type != "array" {

			retMap[typeName] = promptForValue(typeName, v)

			// Simple array type. Prompt for value with delimiter guidance.
		} else if v.Type == "array" && v.Items.Ref == "" {

			// TODO(mierdin) Provide delimiter guidance
			retMap[typeName] = promptForValue(typeName, v)

			// Complex type. Recurse into this function with the new root type
		} else if v.Type == "array" && v.Items.Ref != "" {

			splitSlice := strings.Split(v.Items.Ref, "/")
			subTypeName := splitSlice[len(splitSlice)-1]

			color.HiBlack("Entering subwizard for nested type %s\n", subTypeName)

			var members []interface{}

			i := 0
			for {
				innerMap, _ := schemaWizard(schema, subTypeName, fmt.Sprintf("%s[%d].", typeName, i))
				members = append(members, innerMap)

				if !addMoreToArray(typeName) {
					break
				}

				i++
			}

			retMap[k] = members
		} else {
			// TODO - obviously fix this
			panic("FOOBAR - this should never happen")
		}

	}
	return retMap, nil

}
