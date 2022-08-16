package gitClient

import (
	"fmt"
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

func CloneGitOpsRepo() {

	config := configs.ReadConfig()
	url := "https://github.com/kubefirst/gitops-template"
	directory := fmt.Sprintf("%s/gitops", config.K1FolderPath)

	versionGitOps := viper.GetString("version-gitops")

	log.Println("git clone -b ", versionGitOps, url, directory)

	_, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(versionGitOps),
		SingleBranch:  true,
	})
	if err != nil {
		log.Panicf("error cloning gitops-template repository from github, error is:  %s", err)
	}

	log.Println("downloaded gitops repo from template to directory", config.K1FolderPath, "/gitops")
}

func PushGitopsToSoftServe() {
	cfg := configs.ReadConfig()
	directory := fmt.Sprintf("%s/gitops", cfg.K1FolderPath)

	log.Println("open gitClient repo", directory)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Panicf("error opening the directory ", directory, err)
	}

	log.Println("gitClient remote add origin ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops")
	_, err = repo.CreateRemote(&config.RemoteConfig{
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

	auth, _ := pkg.PublicKey()

	auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	err = repo.Push(&git.PushOptions{
		RemoteName: "soft",
		Auth:       auth,
	})
	if err != nil {
		log.Panicf("error pushing to remote", err)
	}

}

// CloneTemplateRepoWithFallBack - Tries to clone branch, if defined, else try to clone Tag
// In the absence of matching tag/branch function will fail
func CloneTemplateRepoWithFallBack(githubOrg string, repoName string, directory string, branch string, fallbackTag string) error {
	defer viper.WriteConfig()

	repoURL := fmt.Sprintf("https://github.com/%s/%s-template", githubOrg, repoName)

	isMainBranch := true
	isRepoClone := false
	if branch != "main" {
		isMainBranch = false
	}
	//Clone branch if defined
	//Clone tag if defined
	var repo *git.Repository
	var err error
	if branch != "" {
		log.Printf("Trying to clone branch(%s):%s ", branch, repoURL)
		repo, err = git.PlainClone(directory, false, &git.CloneOptions{
			URL:           repoURL,
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			SingleBranch:  true,
		})
		if err != nil {
			log.Printf("error cloning %s-template repository from github %s at branch %s", repoName, err, branch)
		} else {
			isRepoClone = true
			viper.Set(fmt.Sprintf("git.clone.%s.branch", repoName), branch)
		}
	}

	if !isRepoClone && fallbackTag != "" {
		log.Printf("Trying to clone tag(%s):%s ", branch, fallbackTag)
		repo, err = git.PlainClone(directory, false, &git.CloneOptions{
			URL:           repoURL,
			ReferenceName: plumbing.NewTagReferenceName(fallbackTag),
			SingleBranch:  true,
		})
		if err != nil {
			log.Printf("error cloning %s-template repository from github %s at tag %s", repoName, err, fallbackTag)
		} else {
			isRepoClone = true
			viper.Set(fmt.Sprintf("git.clone.%s.tag", repoName), branch)
		}
	}

	if !isRepoClone {
		log.Printf("Error cloning template of repos, code not found on Branch(%s) or Tag(%s) of repo: %s", branch, fallbackTag, repoURL)
		return fmt.Errorf("Error cloning template, No templates found on branch or tag")
	}

	w, _ := repo.Worktree()
	if !isMainBranch {
		branchName := plumbing.NewBranchReferenceName("main")
		headRef, err := repo.Head()
		if err != nil {
			log.Panicf("Error Setting reference: %s, %s", repoName, err)
		}
		ref := plumbing.NewHashReference(branchName, headRef.Hash())
		err = repo.Storer.SetReference(ref)
		if err != nil {
			log.Panicf("error Storing reference: %s, %s", repoName, err)
		}
		err = w.Checkout(&git.CheckoutOptions{Branch: ref.Name()})
	}
	return nil

}
