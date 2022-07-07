package cmd

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kubefirst/nebulous/configs"
	"github.com/spf13/viper"
	"log"
	"time"
)

func hydrateGitlabMetaphorRepo() {
	config := configs.ReadConfig()
	//TODO: Should this be skipped if already executed?
	if !viper.GetBool("create.gitlabmetaphor.cloned") {
		if config.DryRun {
			log.Printf("[#99] Dry-run mode, hydrateGitlabMetaphorRepo skipped.")
			return
		}

		metaphorTemplateDir := fmt.Sprintf("%s/.kubefirst/metaphor", config.HomePath)

		url := "https://github.com/kubefirst/metaphor-template"

		metaphorTemplateRepo, err := git.PlainClone(metaphorTemplateDir, false, &git.CloneOptions{
			URL: url,
		})
		if err != nil {
			log.Panicf("error cloning metaphor-template repo")
		}
		viper.Set("create.gitlabmetaphor.cloned", true)

		detokenize(metaphorTemplateDir)

		viper.Set("create.gitlabmetaphor.detokenized", true)

		// todo make global
		gitlabURL := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.hostedzonename"))
		log.Println("git remote add origin", gitlabURL)
		_, err = metaphorTemplateRepo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "gitlab",
			URLs: []string{fmt.Sprintf("%s/kubefirst/metaphor.git", gitlabURL)},
		})

		w, _ := metaphorTemplateRepo.Worktree()

		log.Println("Committing detokenized metaphor content")
		w.Add(".")
		w.Commit("setting new remote upstream to gitlab", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "kubefirst-bot",
				Email: "kubefirst-bot@kubefirst.com",
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
			log.Panicf("error pushing detokenized metaphor repository to remote at" + gitlabURL)
		}

		viper.Set("create.gitlabmetaphor.pushed", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: hydrateGitlabMetaphorRepo")
	}

}
