package main

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

func getPageFileNames(configjs string) []string {
	r := regexp.MustCompile(`"n":\[".*?"\]`)
	matches := r.FindAllString(configjs, -1)

	for i, match := range matches {
		replacer := strings.NewReplacer(
			"[", "",
			"\"", "",
			"]", "",
		)
		match = strings.Split(match, ":")[1]
		match = replacer.Replace(match)

		matches[i] = match
	}

	return matches
}

func getBookTitle(configjs string) (string, error) {
	r := regexp.MustCompile(`("?(bookConfig.)?bookTitle"?[=]"(.*?)")|"title":"(.*?)"`)
	match := r.FindString(configjs)

	if strings.Contains(match, "=") {
		match = strings.Split(match, "=")[1]
	} else if strings.Contains(match, ":") {
		match = strings.Split(match, ":")[1]
	} else {
		return "", errors.New("could not find book title")
	}
	match = strings.ReplaceAll(match, "\"", "")

	return match, nil
}

func getPageCount(configjs string) (int, error) {
	r := regexp.MustCompile("\"?(bookConfig\\.)?(total)?[Pp]ageCount\"?[=:]\"?\\d+\"?")
	match := r.FindString(configjs)

	if strings.Contains(match, "=") {
		match = strings.Split(match, "=")[1]
	} else if strings.Contains(match, ":") {
		match = strings.Split(match, ":")[1]
	} else {
		return 0, errors.New("could not find page count")
	}

	match = strings.ReplaceAll(match, "\"", "")
	return strconv.Atoi(match)
}
