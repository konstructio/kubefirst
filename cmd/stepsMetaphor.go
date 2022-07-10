package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/viper"
	ssh2 "golang.org/x/crypto/ssh"
)

func pushGitRepo(gitOrigin, repoName string) {

	repoDir := fmt.Sprintf("%s/.kubefirst/%s", home, repoName)
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		log.Panicf("error opening repo %s: %s", repoName, err)
	}

	// todo - fix opts := &git.PushOptions{uniqe, stuff} .Push(opts) ?
	if gitOrigin == "soft" {
		detokenize(repoDir)
		os.RemoveAll(repoDir + "/terraform/base/.terraform")
		os.RemoveAll(repoDir + "/terraform/gitlab/.terraform")
		os.RemoveAll(repoDir + "/terraform/vault/.terraform")
		os.Remove(repoDir + "/terraform/base/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/gitlab/.terraform.lock.hcl")
		commitToRepo(repo, repoName)
		auth, _ := publicKey()

		auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

		err = repo.Push(&git.PushOptions{
			RemoteName: gitOrigin,
			Auth:       auth,
		})
		if err != nil {
			log.Panicf("error pushing detokenized %s repository to remote at %s", repoName, gitOrigin)
		}
		log.Printf("successfully pushed %s to soft-serve", repoName)
	}

	if gitOrigin == "gitlab" {

		auth := &http.BasicAuth{
			Username: "root",
			Password: viper.GetString("gitlab.token"),
		}
		err = repo.Push(&git.PushOptions{
			RemoteName: gitOrigin,
			Auth:       auth,
		})
		if err != nil {
			log.Panicf("error pushing detokenized %s repository to remote at %s", repoName, gitOrigin)
		}
		log.Printf("successfully pushed %s to gitlab", repoName)
	}

	viper.Set(fmt.Sprintf("create.repos.%s.%s.pushed", gitOrigin, repoName), true)
	viper.WriteConfig()
}

func commitToRepo(repo *git.Repository, repoName string) {
	w, _ := repo.Worktree()

	log.Println(fmt.Sprintf("committing detokenized %s kms key id", repoName))
	w.Add(".")
	w.Commit(fmt.Sprintf("committing detokenized %s kms key id", repoName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
}
