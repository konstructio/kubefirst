package cmd

import (
	"fmt"
	"log"
	"os"
	ssh2 "golang.org/x/crypto/ssh"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"path/filepath"
	"strings"
	"github.com/spf13/viper"
	"io/ioutil"
	"time"


)

func cloneGitOpsRepo() {

	url := "https://github.com/kubefirst/gitops-template"
	directory := fmt.Sprintf("%s/.kubefirst/gitops", home)

	log.Println("git clone", url, directory)

	_, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		log.Panicf("reror cloning gitops-template repository from github %s", err)
	}

	log.Println("downloaded gitops repo from template to directory", home, "/.kubefirst/gitops")
}

func pushGitopsToSoftServe() {

	directory := fmt.Sprintf("%s/.kubefirst/gitops", home)

	log.Println("open git repo", directory)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Panicf("error opening the directory ", directory, err)
	}

	log.Println("git remote add origin ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops")
	_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "soft",
		URLs: []string{"ssh://127.0.0.1:8022/gitops"},
	})
	if err != nil {
		log.Panicf("Error creating remote repo: %s", err)
	}
	w, _ := repo.Worktree()

	log.Println("Committing new changes...")
	w.Add(".")
	w.Commit("setting new remote upstream to soft-serve", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})

	auth, _ := publicKey()

	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	err = repo.Push(&git.PushOptions{
		RemoteName: "soft",
		Auth:       auth,
	})
	if err != nil {
		log.Panicf("error pushing to remote", err)
	}

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

	if strings.Contains(path, ".git") || strings.Contains(path, ".terraform") {
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
		domainId := viper.GetString("aws.domainid")
		hostedzonename := viper.GetString("aws.hostedzonename")
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
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_ID>", domainId, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedzonename, -1)
		newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
		newContents = strings.Replace(newContents, "<EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		if kmsKeyId != "" {
			newContents = strings.Replace(newContents, "<KMS_KEY_ID>", kmsKeyId, -1)
		}

		if viper.GetBool("create.terraformapplied.gitlab") {
			newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedzonename, -1)
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
