/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/command"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"
	"github.com/docker/docker/api/types"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	DAEMON_JSON_PATH = "/etc/docker/daemon.json"
	ZIMAOS_MIRROR    = "https://storeproxy.zimaos.com"
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

func UpdateRegistryMirror() {
	if err := createIfNotExistDaemonJsonFile(); err != nil {
		logger.Error("failed to create daemon.json", zap.Error(err))
		return
	}

	var registryAvailable bool
	if content, err := os.ReadFile(DAEMON_JSON_PATH); err == nil {
		mirrors := gjson.Get(string(content), "registry-mirrors").Array()
		if len(mirrors) > 0 {
			var found atomic.Bool
			g := new(errgroup.Group)

			for _, mirror := range mirrors {
				mirrorURL := mirror.String()
				if !strings.HasPrefix(mirrorURL, "https://") {
					continue
				}
				url := mirrorURL
				g.Go(func() error {
					if found.Load() {
						return nil
					}
					logger.Info("checking registry mirror", zap.String("url", url))
					_, err := resty.New().SetTimeout(5 * time.Second).R().Get(strings.TrimSuffix(url, "/") + "/v2/")
					if err == nil {
						found.Store(true)
					}
					return nil
				})
			}
			g.Wait()
			registryAvailable = found.Load()
		}
	}

	if !registryAvailable {
		_, err := resty.New().SetTimeout(5 * time.Second).R().Get("https://registry-1.docker.io/v2/")
		registryAvailable = err == nil
	}

	dockerUnreachableRegion := false
	if resp, err := http.Get("https://ipconfig.io/country"); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if body, err := io.ReadAll(resp.Body); err == nil {
				dockerUnreachableRegion = strings.Contains(string(body), "China")
			}
		}
	} else {
		logger.Error("failed to get ipconfig.io/country", zap.Error(err))
		dockerUnreachableRegion = true
	}

	content, err := os.ReadFile(DAEMON_JSON_PATH)
	if err != nil {
		logger.Error("failed to read daemon.json", zap.Error(err))
		return
	}

	shouldAddMirror := dockerUnreachableRegion && !registryAvailable

	currentMirrors := gjson.Get(string(content), "registry-mirrors").Array()
	var mirrors []string
	hasZimaMirror := false

	for _, mirror := range currentMirrors {
		if strings.Contains(mirror.String(), ZIMAOS_MIRROR) {
			hasZimaMirror = true
			if !shouldAddMirror {
				continue // skip ZIMAOS_MIRROR when we don't want it
			}
		}
		mirrors = append(mirrors, mirror.String())
	}

	needUpdate := false
	if shouldAddMirror && !hasZimaMirror {
		mirrors = append([]string{ZIMAOS_MIRROR}, mirrors...)
		needUpdate = true
	} else if !shouldAddMirror && hasZimaMirror {
		needUpdate = true
	}

	if needUpdate {
		logger.Info("Updating registry mirrors", zap.Bool("shouldAddMirror", shouldAddMirror), zap.Bool("hasZimaMirror", hasZimaMirror))
		if content, err = sjson.SetBytes(content, "registry-mirrors", mirrors); err != nil {
			logger.Error("failed to update registry-mirrors", zap.Error(err))
			return
		}

		if err = os.WriteFile(DAEMON_JSON_PATH, content, 0o644); err != nil {
			logger.Error("failed to write daemon.json", zap.Error(err))
			return
		}

		if err = validateAndRestartDocker(); err != nil {
			logger.Error("failed to restart docker", zap.Error(err))
			return
		}
	}
}

func validateAndRestartDocker() error {
	output, err := command.ExecResultStr("dockerd --validate")
	if strings.Contains(output, "unknown flag: --validate") {
		logger.Info("dockerd --validate not found, skip validation")
	}

	if err != nil || (output != "" && strings.TrimSpace(output) != "configuration OK") {
		logger.Error("Docker configuration validation failed", zap.Error(err), zap.String("output", output))
	}
	if err = systemctl.RestartService("docker.service", true); err != nil {
		logger.Error("failed to restart docker", zap.Error(err))
		return err
	}

	// https://github.com/1Panel-dev/1Panel/blob/dev/backend/app/service/image_repo.go#L98
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	if err := func() error {
		for range ticker.C {
			select {
			case <-ctx.Done():
				cancel()
				return errors.New("the docker service cannot be restarted")
			default:
				stdout, err := command.ExecResultStr("systemctl is-active docker")
				if string(stdout) == "active\n" && err == nil {
					logger.Info("docker restart with new conf successful!")
					return nil
				}
			}
		}
		return nil
	}(); err != nil {
		return err
	}

	return nil
}

// https://github.com/1Panel-dev/1Panel/blob/dev/backend/app/service/docker.go#L242
func createIfNotExistDaemonJsonFile() error {
	if _, err := os.Stat(DAEMON_JSON_PATH); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(DAEMON_JSON_PATH), os.ModePerm); err != nil {
			return err
		}
		var daemonFile *os.File
		daemonFile, err = os.Create(DAEMON_JSON_PATH)
		if err != nil {
			return err
		}
		defer daemonFile.Close()
	}
	return nil
}
