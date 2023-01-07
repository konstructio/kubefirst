package repo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"
	cp "github.com/otiai10/copy"
	"github.com/spf13/viper"
)

// PrepareKubefirstTemplateRepo - Prepare template repo to be used by installer
func PrepareKubefirstTemplateRepo(dryRun bool, config *configs.Config, githubOrg, repoName string, branch string, tag string) {

	log.Info().
		Str("GitHub Organization", githubOrg).
		Str("Repository", repoName).
		Str("Branch", branch).
		Str("Tag", tag).
		Msg("")

	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, PrepareKubefirstTemplateRepo skipped.")
		return
	}
	directory := fmt.Sprintf("%s/%s", config.K1FolderPath, repoName)
	err := gitClient.CloneTemplateRepoWithFallBack(githubOrg, repoName, directory, branch, tag)
	if err != nil {
		log.Panic().Msgf("Error cloning repo with fallback: %s", err)
	}
	viper.Set(fmt.Sprintf("init.repos.%s.cloned", repoName), true)
	viper.WriteConfig()

	log.Info().Msgf("cloned %s-template repository to directory %s/%s", repoName, config.K1FolderPath, repoName)
	if viper.GetString("cloud") == pkg.CloudK3d && !viper.GetBool("github.gitops.hydrated") {
		UpdateForLocalMode(directory)
	}
	if viper.GetString("cloud") == pkg.CloudK3d && strings.Contains(repoName, "metaphor") {
		os.RemoveAll(fmt.Sprintf("%s/.argo", directory))
		os.RemoveAll(fmt.Sprintf("%s/.github", directory))
		opt := cp.Options{
			Skip: func(src string) (bool, error) {
				if strings.HasSuffix(src, ".git") {
					return true, nil
				} else if strings.Index(src, "/.terraform") > 0 {
					return true, nil
				}
				//Add more stuff to be ignored here
				return false, nil

			},
		}
		err = cp.Copy(config.GitOpsLocalRepoPath+"/argo-workflows/.argo", directory+"/.argo", opt)
		if err != nil {
			log.Error().Err(err).Msgf("Error populating argo-workflows .argo/ with local setup: %s", err)
		}
		err = cp.Copy(config.GitOpsLocalRepoPath+"/argo-workflows/.github", directory+"/.github", opt)
		if err != nil {
			log.Error().Err(err).Msgf("Error populating argo-workflows with .github/ with local setup: %s", err)
		}
	}

	log.Info().Msgf("detokenizing %s/%s", config.K1FolderPath, repoName)
	pkg.Detokenize(directory)
	log.Info().Msgf("detokenization of %s/%s complete", config.K1FolderPath, repoName)

	viper.Set(fmt.Sprintf("init.repos.%s.detokenized", repoName), true)
	viper.WriteConfig()

	repo, err := git.PlainOpen(directory)

	if viper.GetString("gitprovider") == "github" {
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")

		url := fmt.Sprintf("https://%s/%s/%s", githubHost, githubOwner, repoName)
		log.Info().Msgf("git remote add github %s", url)
		_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "github",
			URLs: []string{url},
		})
	} else {
		domain := viper.GetString("aws.hostedzonename")
		log.Info().Msg("creating git remote gitlab")
		log.Info().Msgf("git remote add gitlab at url %s", fmt.Sprintf("https://gitlab.%s/kubefirst/%s.git", domain, repoName))
		if err != nil {
			log.Panic().Msgf("error opening the directory %s: %s", directory, err)
		}

		_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "gitlab",
			URLs: []string{fmt.Sprintf("https://gitlab.%s/kubefirst/%s.git", domain, repoName)},
		})
		if repoName == "gitops" {
			log.Info().Msg("creating git remote ssh://127.0.0.1:8022/gitops")
			_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
				Name: "soft",
				URLs: []string{"ssh://127.0.0.1:8022/gitops"},
			})
		}
	}

	if err != nil {
		log.Panic().Msgf("Error creating remote %s for repo: %s - %s", viper.GetString("git.mode"), repoName, err)
	}

	w, _ := repo.Worktree()

	log.Info().Msgf("committing detokenized %s content", repoName)
	err = gitClient.GitAddWithFilter(viper.GetString("cloud"), repoName, w)
	if err != nil {
		log.Error().Err(err).Msg("error getting worktree status")
	}
	w.Commit(fmt.Sprintf("[ci skip] committing detokenized %s content", repoName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
	viper.WriteConfig()
}

// UpdateForLocalMode - Tweak for local install on templates
func UpdateForLocalMode(directory string) error {
	log.Info().Msgf("Working Directory: %s", directory)
	//Tweak folder
	os.RemoveAll(directory + "/components")
	os.RemoveAll(directory + "/registry")
	os.RemoveAll(directory + "/terraform")
	os.RemoveAll(directory + "/validation")
	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			//Add more stuff to be ignored here
			return false, nil

		},
	}
	err := cp.Copy(directory+"/localhost", directory, opt)
	if err != nil {
		log.Error().Err(err).Msg("Error populating gitops with local setup")
		return err
	}
	os.RemoveAll(directory + "/localhost")
	return nil
}
