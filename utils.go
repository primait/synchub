package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
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

func mergeStruct(to interface{}, from interface{}) error {
	byte1, err := json.Marshal(to)
	if err != nil {
		return err
	}
	byte2, err := json.Marshal(from)
	if err != nil {
		return err
	}
	map1 := make(map[string]interface{})
	err = json.Unmarshal(byte1, &map1)
	if err != nil {
		return err
	}
	map2 := make(map[string]interface{})
	err = json.Unmarshal(byte2, &map2)
	if err != nil {
		return err
	}
	for k, v := range map2 {
		map1[k] = v
	}
	byteDest, err := json.Marshal(map1)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteDest, to)
	return err
}


func stringInSlice(str string, list []string) bool {
	for _, elem := range list {
		if elem == str {
			return true
		}
	}
	return false
}
