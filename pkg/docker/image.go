/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/docker/docker/client"
)

func IsImageStale(imageName string, currentImageID string) (bool, string, error) {
	ctx := context.Background()

	if err := PullNewImage(ctx, imageName); err != nil {
		return false, currentImageID, err
	}

	return HasNewImage(ctx, imageName, currentImageID)
}

func PullNewImage(ctx context.Context, imageName string) error {
	if strings.HasPrefix(imageName, "sha256:") {
		return fmt.Errorf("container uses a pinned image, and cannot be updated")
	}

	opts, err := GetPullOptions(imageName)
	if err != nil {
		return err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()

	imageInfo, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return err
	}

	if match, err := CompareDigest(imageName, imageInfo.RepoDigests, opts.RegistryAuth); err != nil {
		// do nothing
	} else if match {
		return nil
	}

	response, err := cli.ImagePull(ctx, imageName, opts)
	if err != nil {
		return err
	}
	defer response.Close()

	if _, err := ioutil.ReadAll(response); err != nil {
		return err
	}

	return nil
}

func HasNewImage(ctx context.Context, imageName string, currentImageID string) (bool, string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false, currentImageID, err
	}
	defer cli.Close()

	newImageInfo, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return false, currentImageID, err
	}

	newImageID := newImageInfo.ID
	if newImageID != currentImageID {
		return false, currentImageID, nil
	}

	return true, newImageID, nil
}
