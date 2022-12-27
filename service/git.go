package service

import (
	"crypto/md5" // nolint: gosec
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"

	"github.com/go-git/go-git/v5"
)

type GitService struct {
	appStoreGitRepos map[string]string
}

func NewGitService() *GitService {
	repos := make(map[string]string)

	// TODO - package the latest app store along with other packages

	// TODO - select last successful app store dir

	// TODO - run in parallel
	for _, repoURL := range config.ServerInfo.AppStoreList {
		if _, err := url.Parse(repoURL); err != nil {
			logger.Error("invalid app store url", zap.Error(err), zap.String("url", repoURL))
			continue
		}

		dir := repoDir(repoURL)

		if err := file.IsNotExistMkDir(dir); err != nil {
			logger.Error("create app store dir failed", zap.Error(err), zap.String("dir", dir))
			continue
		}

		if r, err := getRepo(dir); err != nil {
			logger.Info("not a valid repo - trying to clean and clone", zap.String("repo", repoURL))

			if err := cloneRepo(repoURL, dir, true); err != nil {
				logger.Error("clone app store repo failed", zap.Error(err), zap.String("repo", repoURL))
				continue
			}

			logger.Info("clone app store repo success", zap.String("repo", repoURL))
		} else {

			// TODO - start background job to pull repo periodically
			if err := pullRepo(r); err != nil {
				logger.Info("pull app store repo failed - trying to clean and clone", zap.Error(err), zap.String("repo", repoURL))
				continue
			}

			logger.Info("pull app store repo success", zap.String("repo", repoURL))
		}

		repos[repoURL] = dir
	}

	return &GitService{
		appStoreGitRepos: repos,
	}
}

func repoDir(repo string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(repo))) //nolint: gosec
	return filepath.Join(config.AppInfo.AppStorePath, hash)
}

func getRepo(dir string) (*git.Repository, error) {
	return git.PlainOpen(dir)
}

func pullRepo(r *git.Repository) error {
	w, err := r.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		Force:      true,
		Depth:      1,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}

func cloneRepo(repoURL, dir string, clean bool) error {
	if clean {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	if _, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Depth:    1,
	}); err != nil {
		return err
	}

	return nil
}
