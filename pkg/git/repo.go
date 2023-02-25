package git

import (
	"errors"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
)

var (
	ErrEmptyRepo           = errors.New("empty repo")
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
)

func Clone(gitURL, dir string) error {
	if err := file.RMDir(dir); err != nil {
		return err
	}

	if err := file.IsNotExistMkDir(dir); err != nil {
		return err
	}

	if _, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:      gitURL,
		Progress: os.Stdout,
		Depth:    1,
	}); err != nil {
		return err
	}

	return nil
}

func Pull(dir string) error {
	r, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{
		Progress:     os.Stdout,
		Force:        true,
		Depth:        1,
		SingleBranch: true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}

func ValidateGitURL(gitURL string) error {
	if !isHTTPBased(gitURL) {
		return ErrUnsupportedProtocol
	}

	storage := memory.NewStorage()

	remote := git.NewRemote(storage, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitURL},
	})

	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return err
	}

	if len(refs) == 0 {
		return ErrEmptyRepo
	}

	return nil
}

func WorkDir(gitURL string, baseDir string) (string, error) {
	gitURL = strings.TrimSuffix(gitURL, ".git")
	gitURL = strings.TrimRight(gitURL, "/")
	gitURL = strings.ToLower(gitURL)

	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return "", err
	}

	return path.Join(baseDir, parsedURL.Host, parsedURL.Path), nil
}

func isHTTPBased(url string) bool {
	return regexp.MustCompile(`^https?://`).MatchString(url)
}
