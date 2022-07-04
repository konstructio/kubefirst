package cmd

import (
	"fmt"
	"log"
	"github.com/spf13/viper"
	"time"
	"github.com/go-git/go-git/v5"	
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

func hydrateGitlabMetaphorRepo() {
	//TODO: Should this be skipped if already executed?
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, hydrateGitlabMetaphorRepo skipped.")
		return
	}
	metaphorTemplateDir := fmt.Sprintf("%s/.kubefirst/metaphor", home)

	url := "https://github.com/kubefirst/metaphor-template"

	metaphorTemplateRepo, err := git.PlainClone(metaphorTemplateDir, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		panic("error cloning metaphor-template repo")
	}

	detokenize(metaphorTemplateDir)

	// todo make global
	domainName := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname"))
	log.Println("git remote add origin", domainName)
	_, err = metaphorTemplateRepo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "gitlab",
		URLs: []string{fmt.Sprintf("%s/kubefirst/metaphor.git", domainName)},
	})

	w, _ := metaphorTemplateRepo.Worktree()

	log.Println("Committing new changes...")
	w.Add(".")
	w.Commit("setting new remote upstream to gitlab", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: installerEmail,
			When:  time.Now(),
		},
	})

	err = metaphorTemplateRepo.Push(&git.PushOptions{
		RemoteName: "gitlab",
		Auth: &gitHttp.BasicAuth{
			Username: "root",
			Password: viper.GetString("gitlab.token"),
		},
	})
	if err != nil {
		log.Println("error pushing to remote", err)
	}

}