/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"fmt"
	"strings"

	url2 "net/url"

	ref "github.com/docker/distribution/reference"
)

// BuildManifestURL from raw image data
func BuildManifestURL(imageName string) (string, error) {
	normalizedName, err := ref.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", err
	}

	host, err := NormalizeRegistry(normalizedName.String())
	img, tag := ExtractImageAndTag(strings.TrimPrefix(imageName, host+"/"))

	if err != nil {
		return "", err
	}
	img = GetScopeFromImageName(img, host)

	if !strings.Contains(img, "/") {
		img = "library/" + img
	}
	url := url2.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", img, tag),
	}
	return url.String(), nil
}

// ExtractImageAndTag from a concatenated string
func ExtractImageAndTag(imageName string) (string, string) {
	var img string
	var tag string

	if strings.Contains(imageName, ":") {
		parts := strings.Split(imageName, ":")
		if len(parts) > 2 {
			img = parts[0]
			tag = strings.Join(parts[1:], ":")
		} else {
			img = parts[0]
			tag = parts[1]
		}
	} else {
		img = imageName
		tag = "latest"
	}
	return img, tag
}
