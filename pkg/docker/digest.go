/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// RegistryCredentials is a credential pair used for basic auth
type RegistryCredentials struct {
	Username string
	Password string // usually a token rather than an actual password
}

// ContentDigestHeader is the key for the key-value pair containing the digest header
const ContentDigestHeader = "Docker-Content-Digest"

// CompareDigest ...
func CompareDigest(imageName string, repoDigests []string, registryAuth string) (bool, error) {
	var digest string

	registryAuth = TransformAuth(registryAuth)
	challenge, err := GetChallenge(imageName)
	if err != nil {
		return false, err
	}

	token, err := GetToken(challenge, registryAuth, imageName)
	if err != nil {
		return false, err
	}

	digestURL, err := BuildManifestURL(imageName)
	if err != nil {
		return false, err
	}

	if digest, err = GetDigest(digestURL, token); err != nil {
		return false, err
	}

	for _, dig := range repoDigests {
		localDigest := strings.Split(dig, "@")[1]

		if localDigest == digest {
			return true, nil
		}
	}

	return false, nil
}

// TransformAuth from a base64 encoded json object to base64 encoded string
func TransformAuth(registryAuth string) string {
	b, _ := base64.StdEncoding.DecodeString(registryAuth)
	credentials := &RegistryCredentials{}
	_ = json.Unmarshal(b, credentials)

	if credentials.Username != "" && credentials.Password != "" {
		ba := []byte(fmt.Sprintf("%s:%s", credentials.Username, credentials.Password))
		registryAuth = base64.StdEncoding.EncodeToString(ba)
	}

	return registryAuth
}

// GetDigest from registry using a HEAD request to prevent rate limiting
func GetDigest(url string, token string) (string, error) {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		DisableKeepAlives:     true,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		Proxy:                 http.ProxyFromEnvironment,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		TLSHandshakeTimeout:   10 * time.Second,
	}
	client := &http.Client{Transport: tr}

	req, _ := http.NewRequest(http.MethodHead, url, nil)
	// req.Header.Set("User-Agent", userAgent) - confirm if this is needed

	if token == "" {
		return "", errors.New("could not fetch token")
	}

	req.Header.Add("Authorization", token)
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		wwwAuthHeader := res.Header.Get("www-authenticate")
		if wwwAuthHeader == "" {
			wwwAuthHeader = "not present"
		}
		return "", fmt.Errorf("registry responded to head request with %q, auth: %q", res.Status, wwwAuthHeader)
	}
	return res.Header.Get(ContentDigestHeader), nil
}
