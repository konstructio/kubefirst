/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitClient

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"

	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func Clone(gitRef, repoLocalPath, repoURL string) (*git.Repository, error) {

	// kubefirst tags do not contain a `v` prefix, to use the library requires the v to be valid
	isSemVer := semver.IsValid("v" + gitRef)

	var refName plumbing.ReferenceName

	if isSemVer {
		refName = plumbing.NewTagReferenceName(gitRef)
	} else {
		refName = plumbing.NewBranchReferenceName(gitRef)
	}

	repo, err := git.PlainClone(repoLocalPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: refName,
		SingleBranch:  true,
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func CloneRefSetMain(gitRef, repoLocalPath, repoURL string) (*git.Repository, error) {

	log.Info().Msgf("cloning url: %s - git ref: %s", repoURL, gitRef)

	repo, err := Clone(gitRef, repoLocalPath, repoURL)
	if err != nil {
		return nil, err
	}

	if gitRef != "main" {
		repo, err = SetRefToMainBranch(repo)
		if err != nil {
			return nil, fmt.Errorf("error setting main branch from git ref: %s", gitRef)
		}

		// remove old git ref
		err = repo.Storer.RemoveReference(plumbing.NewBranchReferenceName(gitRef))
		if err != nil {
			return nil, fmt.Errorf("error removing previous git ref: %s", err)
		}
	}
	return repo, nil
}

// SetRefToMainBranch sets the provided gitRef (branch or tag) to the main branch
func SetRefToMainBranch(repo *git.Repository) (*git.Repository, error) {
	w, _ := repo.Worktree()
	branchName := plumbing.NewBranchReferenceName("main")
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("error Setting reference: %s", err)
	}

	ref := plumbing.NewHashReference(branchName, headRef.Hash())
	err = repo.Storer.SetReference(ref)
	if err != nil {
		return nil, fmt.Errorf("error Storing reference: %s", err)
	}

	err = w.Checkout(&git.CheckoutOptions{Branch: ref.Name()})
	if err != nil {
		return nil, fmt.Errorf("error checking out main: %s", err)
	}
	return repo, nil
}

func AddRemote(newGitRemoteURL, remoteName string, repo *git.Repository) error {

	log.Info().Msgf("git remote add %s %s", remoteName, newGitRemoteURL)
	_, err := repo.CreateRemote(&gitConfig.RemoteConfig{
		Name: remoteName,
		URLs: []string{newGitRemoteURL},
	})
	if err != nil {
		log.Info().Msgf("Error creating remote %s at: %s", remoteName, newGitRemoteURL)
		return err
	}
	return nil
}

func Commit(repo *git.Repository, commitMsg string) error {
	w, err := repo.Worktree()
	if err != nil {
		log.Info().Msgf("error getting worktree: %s", err)
		return err
	}

	log.Info().Msg(commitMsg)
	status, err := w.Status()
	if err != nil {
		log.Info().Msgf("error getting worktree status: %s", err)
		return err
	}

	for file, _ := range status {
		_, err = w.Add(file)
		if err != nil {
			log.Info().Msgf("error getting worktree status: %s", err)
			return err
		}
	}
	w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kbot",
			Email: "kbot@kubefirst.com",
			When:  time.Now(),
		},
	})
	return nil
}
