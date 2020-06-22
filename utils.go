package main

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func askForConfirmation(message string) bool {
	fmt.Println(message)
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
		return askForConfirmation(message)
	}
}

func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

func slug(s string) string {
	re := regexp.MustCompile("[^a-z0-9]+")
	return strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func mergeOverwrite(to, from, dst interface{}) error {
	toMap := structs.Map(to)
	toStruct := structs.New(to)
	fromMap := structs.Map(from)
	fromStruct := structs.New(from)
	for k, v := range fromMap {
		_, ok := toMap[k]
		if !ok {
			return errors.Errorf("no key: %s", k)
		}
		toField := toStruct.Field(k)
		fromField := fromStruct.Field(k)
		if overwriteable(toField, fromField) {
			toMap[k] = v
		}
	}
	if err := mapstructure.Decode(toMap, dst); err != nil {
		return errors.Wrap(err, "faield to decode")
	}
	return nil
}

func overwriteable(to, from *structs.Field) bool {
	if to.Kind().String() == "bool" {
		return true
	}
	if reflect.TypeOf(to).Kind().String() == "struct" {
		return false
	}
	switch {
	case to.IsZero() && !from.IsZero():
		return true
	case !to.IsZero() && !from.IsZero():
		return true
	case !to.IsZero() && from.IsZero():
		return false
	}
	return true
}