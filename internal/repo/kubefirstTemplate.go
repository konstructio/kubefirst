package repo

import (
	"fmt"
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

//PrepareKubefirstTemplateRepo - Prepare template repo to be used by installer
func PrepareKubefirstTemplateRepo(dryRun bool, config *configs.Config, githubOrg, repoName string, branch string, tag string) {

	if dryRun {
		log.Printf("[#99] Dry-run mode, PrepareKubefirstTemplateRepo skipped.")
		return
	}
	directory := fmt.Sprintf("%s/%s", config.K1FolderPath, repoName)
	err := gitClient.CloneTemplateRepoWithFallBack(githubOrg, repoName, directory, branch, tag)
	if err != nil {
		log.Panicf("Error cloning repo with fallback: %s", err)
	}
	viper.Set(fmt.Sprintf("init.repos.%s.cloned", repoName), true)
	viper.WriteConfig()

	log.Printf("cloned %s-template repository to directory %s/%s", repoName, config.K1FolderPath, repoName)

	log.Printf("detokenizing %s/%s", config.K1FolderPath, repoName)
	pkg.Detokenize(directory)
	log.Printf("detokenization of %s/%s complete", config.K1FolderPath, repoName)

	viper.Set(fmt.Sprintf("init.repos.%s.detokenized", repoName), true)
	viper.WriteConfig()

	repo, err := git.PlainOpen(directory)

	if viper.GetBool("github.enabled") {
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")

		url := fmt.Sprintf("https://%s/%s/%s", githubHost, githubOwner, repoName)
		log.Printf("git remote add github %s", url)
		_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "github",
			URLs: []string{url},
		})
	} else {
		domain := viper.GetString("aws.hostedzonename")
		log.Printf("creating git remote gitlab")
		log.Println("git remote add gitlab at url ", fmt.Sprintf("https://gitlab.%s/kubefirst/%s.git", domain, repoName))
		if err != nil {
			log.Panicf("error opening the directory %s:  %s", directory, err)
		}

		_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "gitlab",
			URLs: []string{fmt.Sprintf("https://gitlab.%s/kubefirst/%s.git", domain, repoName)},
		})
		if repoName == "gitops" {
			log.Println("creating git remote ssh://127.0.0.1:8022/gitops")
			_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
				Name: "soft",
				URLs: []string{"ssh://127.0.0.1:8022/gitops"},
			})
		}
	}

	if err != nil {
		log.Panicf("Error creating remote %s for repo: %s - %s", viper.GetString("git.mode"), repoName, err)
	}

	w, _ := repo.Worktree()

	log.Printf("committing detokenized %s content", repoName)
	status, err := w.Status()
	if err != nil {
		log.Println("error getting worktree status", err)
	}

	for file, s := range status {
		log.Printf("the file is %s the status is %v", file, s.Worktree)
		_, err = w.Add(file)
		if err != nil {
			log.Println("error getting worktree status", err)
		}
	}
	w.Commit(fmt.Sprintf("committing detokenized %s content", repoName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
	viper.WriteConfig()
}
