/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"github.com/docker/docker/api/types"
)

// GetPullOptions creates a struct with all options needed for pulling images from a registry
func GetPullOptions(imageName string) (types.ImagePullOptions, error) {
	auth, err := EncodedAuth(imageName)
	if err != nil {
		return types.ImagePullOptions{}, err
	}

	if auth == "" {
		return types.ImagePullOptions{}, nil
	}

	return types.ImagePullOptions{
		RegistryAuth:  auth,
		PrivilegeFunc: func() (string, error) { return "", nil },
	}, nil
}
