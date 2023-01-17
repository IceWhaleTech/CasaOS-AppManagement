package docker

import (
	"context"

	"github.com/docker/docker/client"
)

func IsDaemonRunning() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer cli.Close()

	_, err = cli.Ping(context.Background())
	return err == nil
}

func CurrentArchitecture() (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	defer cli.Close()

	ver, err := cli.ServerVersion(context.Background())
	if err != nil {
		return "", err
	}

	return ver.Arch, nil
}
