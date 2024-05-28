/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// RegistryCredentials is a credential pair used for basic auth
type RegistryCredentials struct {
	Username string
	Password string // usually a token rather than an actual password
}

// ContentDigestHeader is the key for the key-value pair containing the digest header
const ContentDigestHeader = "Docker-Content-Digest"

// CompareDigest ...
func CompareDigest(imageName string, repoDigests []string) (bool, error) {
	var digest string

	token, url, err := tokenAndURL(imageName)
	if err != nil {
		return false, err
	}

	if digest, err = GetDigest(url, token); err != nil {
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
	if token == "" {
		return "", errors.New("could not fetch token")
	}

	req, _ := http.NewRequest(http.MethodHead, url, nil)
	addDefaultHeaders(&req.Header, token)

	res, err := httpClient().Do(req)
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

func GetManifest(ctx context.Context, imageName string) (interface{}, string, error) {
	token, url, err := tokenAndURL(imageName)
	if err != nil {
		return nil, "", err
	}

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(ctx)
	addDefaultHeaders(&req.Header, token)

	res, err := httpClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("registry responded to head request with %q", res.Status)
	}

	contentType := res.Header.Get("Content-Type")

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var baseManifest manifest.Versioned
	if err := json.Unmarshal(buf, &baseManifest); err != nil {
		return nil, contentType, fmt.Errorf("not a manifest content: %w", err)
	}

	manifest, ok := map[string]interface{}{
		schema1.MediaTypeSignedManifest:    schema1.SignedManifest{},
		schema2.MediaTypeManifest:          schema2.Manifest{},
		manifestlist.MediaTypeManifestList: manifestlist.ManifestList{},
		v1.MediaTypeImageIndex:             manifestlist.ManifestList{},
	}[contentType]

	if !ok {
		return nil, contentType, fmt.Errorf("unknown content type: %s", contentType)
	}

	if err := json.Unmarshal(buf, &manifest); err != nil {
		return nil, "", err
	}

	return manifest, contentType, nil
}

func tokenAndURL(imageName string) (string, string, error) {
	opts, err := GetPullOptions(imageName)
	if err != nil {
		return "", "", err
	}

	registryAuth := TransformAuth(opts.RegistryAuth)
	challenge, err := GetChallenge(imageName)
	if err != nil {
		return "", "", err
	}

	token, err := GetToken(challenge, registryAuth, imageName)
	if err != nil {
		return "", "", err
	}

	url, err := BuildManifestURL(imageName)
	if err != nil {
		return "", "", err
	}

	return token, url, nil
}

func httpClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
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
	}}
}

func addDefaultHeaders(header *http.Header, token string) {
	header.Add("Authorization", token)
	// header.Add("Accept", schema2.MediaTypeManifest)
	header.Add("Accept", manifestlist.MediaTypeManifestList)
	// header.Add("Accept", schema1.MediaTypeManifest)
	header.Add("Accept", v1.MediaTypeImageIndex)
}
