/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func Image(ctx context.Context, imageName string) (*types.ImageInspect, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	imageInfo, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, err
	}

	return &imageInfo, nil
}

func PullImage(ctx context.Context, imageName string, handleOut func(io.ReadCloser)) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	opts, err := GetPullOptions(imageName)
	if err != nil {
		return err
	}

	out, err := cli.ImagePull(ctx, imageName, opts)
	if err != nil {
		return err
	}
	defer out.Close()

	if handleOut != nil {
		handleOut(out)
	} else {
		if _, err := io.ReadAll(out); err != nil {
			return err
		}
	}

	return nil
}

func HasNewImage(ctx context.Context, imageName string, currentImageID string) (bool, string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, currentImageID, err
	}
	defer cli.Close()

	newImageInfo, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return false, currentImageID, err
	}

	newImageID := newImageInfo.ID
	if newImageID == currentImageID {
		return false, currentImageID, nil
	}

	return true, newImageID, nil
}
