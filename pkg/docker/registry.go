/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"

	"github.com/docker/docker/api/types/image"
)

// GetPullOptions creates a struct with all options needed for pulling images from a registry
func GetPullOptions(imageName string) (image.PullOptions, error) {
	auth, err := EncodedAuth(imageName)
	if err != nil {
		return image.PullOptions{}, err
	}

	if auth == "" {
		return image.PullOptions{}, nil
	}

	return image.PullOptions{
		RegistryAuth:  auth,
		PrivilegeFunc: func(context.Context) (string, error) { return "", nil },
	}, nil
}
