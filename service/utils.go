package service

import (
	"regexp"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"gopkg.in/yaml.v3"
)

var nonAlphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)

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

func GenerateYAMLFromComposeApp(compose ComposeApp) ([]byte, error) {
	// to duplicate Specify Chars
	for _, service := range compose.Services {
		// it should duplicate all values that contains $. But for now, we only duplicate the env values
		for key, value := range service.Environment {
			if strings.ContainsAny(*value, "$") {
				service.Environment[key] = utils.Ptr(strings.Replace(*value, "$", "$$", -1))
			}
		}
	}
	return yaml.Marshal(compose)
}
