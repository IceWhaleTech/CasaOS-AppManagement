package git

import (
	"errors"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
)

var ErrEmptyRepo = errors.New("empty repo")

func ValidateGitURL(url string) error {
	storage := memory.NewStorage()

	remote := git.NewRemote(storage, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
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
