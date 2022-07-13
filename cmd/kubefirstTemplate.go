package cmd

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func prepareKubefirstTemplateRepo(config *configs.Config, githubOrg, repoName string, branch string) {

	if branch == "" {
		branch = "main"
	}

	repoUrl := fmt.Sprintf("https://github.com/%s/%s-template", githubOrg, repoName)
	directory := fmt.Sprintf("%s/%s", config.K1srtFolderPath, repoName)
	log.Println("git clone", repoUrl, directory)
	log.Println("git clone -b ", branch, repoUrl, directory)

	repo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:           repoUrl,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		log.Panicf("error cloning %s-template repository from github %s", repoName, err)
	}
	viper.Set(fmt.Sprintf("init.repos.%s.cloned", repoName), true)
	viper.WriteConfig()

	log.Printf("cloned %s-template repository to directory %s/%s", repoName, config.K1srtFolderPath, repoName)

	log.Printf("detokenizing %s/%s", config.K1srtFolderPath, repoName)
	detokenize(directory)
	log.Printf("detokenization of %s/%s complete", config.K1srtFolderPath, repoName)

	viper.Set(fmt.Sprintf("init.repos.%s.detokenized", repoName), true)
	viper.WriteConfig()

	domain := viper.GetString("aws.hostedzonename")
	log.Printf("creating git remote gitlab")
	log.Println("git remote add gitlab at url ", fmt.Sprintf("https://gitlab.%s/kubefirst/%s.git", domain, repoName))
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

	w, _ := repo.Worktree()

	log.Println(fmt.Sprintf("committing detokenized %s content", repoName))
	w.Add(".")
	w.Commit(fmt.Sprintf("committing detokenized %s content", repoName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
	viper.WriteConfig()
}

func detokenize(path string) {

	err := filepath.Walk(path, detokenizeDirectory)
	if err != nil {
		panic(err)
	}
}

func detokenizeDirectory(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil //
	}

	if strings.Contains(path, ".git/") || strings.Contains(path, ".terraform") {
		return nil
	}

	matched, err := filepath.Match("*", fi.Name())

	if err != nil {
		panic(err)
	}

	if matched {
		read, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		// todo should detokenize be a switch statement based on a value found in viper?
		gitlabConfigured := viper.GetBool("gitlab.keyuploaded")

		newContents := ""

		if gitlabConfigured {
			newContents = strings.Replace(string(read), "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename")), -1)
		} else {
			newContents = strings.Replace(string(read), "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops.git", "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", -1)
		}

		botPublicKey := viper.GetString("botpublickey")
		hostedZoneId := viper.GetString("aws.hostedzoneid")
		hostedZoneName := viper.GetString("aws.hostedzonename")
		bucketStateStore := viper.GetString("bucket.state-store.name")
		bucketArgoArtifacts := viper.GetString("bucket.argo-artifacts.name")
		bucketGitlabBackup := viper.GetString("bucket.gitlab-backup.name")
		bucketChartmuseum := viper.GetString("bucket.chartmuseum.name")
		region := viper.GetString("aws.region")
		adminEmail := viper.GetString("adminemail")
		awsAccountId := viper.GetString("aws.accountid")
		kmsKeyId := viper.GetString("vault.kmskeyid")

		newContents = strings.Replace(newContents, "<SOFT_SERVE_INITIAL_ADMIN_PUBLIC_KEY>", strings.TrimSpace(botPublicKey), -1)
		newContents = strings.Replace(newContents, "<TF_STATE_BUCKET>", bucketStateStore, -1)
		newContents = strings.Replace(newContents, "<ARGO_ARTIFACT_BUCKET>", bucketArgoArtifacts, -1)
		newContents = strings.Replace(newContents, "<GITLAB_BACKUP_BUCKET>", bucketGitlabBackup, -1)
		newContents = strings.Replace(newContents, "<CHARTMUSEUM_BUCKET>", bucketChartmuseum, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_ID>", hostedZoneId, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedZoneName, -1)
		newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
		newContents = strings.Replace(newContents, "<EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		if kmsKeyId != "" {
			newContents = strings.Replace(newContents, "<KMS_KEY_ID>", kmsKeyId, -1)
		}

		if viper.GetBool("create.terraformapplied.gitlab") {
			newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedZoneName, -1)
			newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
			newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		}

		err = ioutil.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			panic(err)
		}

	}

	return nil
}
