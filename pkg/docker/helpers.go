/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"fmt"
	url2 "net/url"
	"time"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/mitchellh/mapstructure"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
)

var Cache *cache.Cache

func init() {
	Cache = cache.New(5*time.Minute, 60*time.Second)
}

// ConvertToHostname strips a url from everything but the hostname part
func ConvertToHostname(url string) (string, string, error) {
	urlWithSchema := fmt.Sprintf("x://%s", url)
	u, err := url2.Parse(urlWithSchema)
	if err != nil {
		return "", "", err
	}
	hostName := u.Hostname()
	port := u.Port()

	return hostName, port, err
}

// NormalizeRegistry makes sure variations of DockerHubs registry
func NormalizeRegistry(registry string) (string, error) {
	hostName, port, err := ConvertToHostname(registry)
	if err != nil {
		return "", err
	}

	if hostName == "registry-1.docker.io" || hostName == "docker.io" {
		hostName = "index.docker.io"
	}

	if port != "" {
		return fmt.Sprintf("%s:%s", hostName, port), nil
	}
	return hostName, nil
}

func GetArchitectures(imageName string, noCache bool) ([]string, error) {
	cacheKey := imageName + ":architectures"
	if !noCache && Cache != nil {
		if cached, ok := Cache.Get(cacheKey); ok {
			if archs, ok := cached.([]string); ok {
				return archs, nil
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manfest, contentType, err := GetManifest(ctx, imageName)
	if err != nil {
		return nil, err
	}

	fmt.Printf("got manifest - image: %s, contentType: %s", imageName, contentType)

	var architectures []string

	architectures, err = tryGetArchitecturesFromManifestList(manfest)
	if err != nil {
		fmt.Printf("failed to get architectures from manifest list: %v", err)
	}

	if len(architectures) == 0 {
		architectures, err = tryGetArchitecturesFromV1SignedManifest(manfest)
		if err != nil {
			fmt.Printf("failed to get architectures from v1 signed manifest: %v", err)
		}
	}

	if Cache != nil && len(architectures) > 0 {
		Cache.Set(cacheKey, architectures, 4*time.Hour)
	} else {
		fmt.Println("WARNING: cache is not initialized - will still be getting container image manifest from network next time.")
	}

	return architectures, nil
}

func tryGetArchitecturesFromManifestList(manifest interface{}) ([]string, error) {
	var listManifest manifestlist.ManifestList
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &listManifest, Squash: true})
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(manifest); err != nil {
		return nil, err
	}

	architectures := []string{}
	for _, platform := range listManifest.Manifests {
		if platform.Platform.Architecture == "" || platform.Platform.Architecture == "unknown" {
			continue
		}

		architectures = append(architectures, platform.Platform.Architecture)
	}

	architectures = lo.Uniq(architectures)

	return architectures, nil
}

func tryGetArchitecturesFromV1SignedManifest(manifest interface{}) ([]string, error) {
	var signedManifest schema1.SignedManifest
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &signedManifest, Squash: true})
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(manifest); err != nil {
		return nil, err
	}

	if signedManifest.Architecture == "" || signedManifest.Architecture == "unknown" {
		return []string{"amd64"}, nil // bad assumption, but works for 99% of the cases
	}

	return []string{signedManifest.Architecture}, nil
}
