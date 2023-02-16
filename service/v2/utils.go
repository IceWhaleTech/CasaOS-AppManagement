package v2

import (
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

var (
	nonAlphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)
	json            = jsoniter.ConfigCompatibleWithStandardLibrary
)

func Standardize(text string) string {
	if text == "" {
		return "unknown"
	}

	result := strings.ToLower(text)

	// Replace any non-alphanumeric characters with a single hyphen
	result = nonAlphaNumeric.ReplaceAllString(result, "-")

	for strings.Contains(result, "--") {
		result = strings.Replace(result, "--", "-", -1)
	}

	// Remove any leading or trailing hyphens
	result = strings.Trim(result, "-")

	return result
}
