package gitClient

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	cp "github.com/otiai10/copy"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

// CloneRepoAndDetokenizeTemplate - clone repo using CloneRepoAndDetokenizeTemplate that uses fallback rule to try to capture version
func CloneRepoAndDetokenizeTemplate(githubOwner, repoName, folderName string, branch string, tag string) (string, error) {
	config := configs.ReadConfig()

	directory := fmt.Sprintf("%s/%s", config.K1FolderPath, folderName)
	err := os.RemoveAll(directory)
	if err != nil {
		log.Println("Error removing dir(expected if dir not present):", err)
	}

	err = CloneTemplateRepoWithFallBack(githubOwner, repoName, directory, branch, tag)
	if err != nil {
		log.Panicf("Error cloning repo with fallback: %s", err)
	}
	if err != nil {
		log.Printf("error cloning %s repository from github %s", folderName, err)
		return directory, err
	}
	viper.Set(fmt.Sprintf("init.repos.%s.cloned", folderName), true)
	viper.WriteConfig()

	log.Printf("cloned %s-template repository to directory %s/%s", folderName, config.K1FolderPath, folderName)

	log.Printf("detokenizing %s/%s", config.K1FolderPath, folderName)
	pkg.Detokenize(directory)
	log.Printf("detokenization of %s/%s complete", config.K1FolderPath, folderName)

	viper.Set(fmt.Sprintf("init.repos.%s.detokenized", folderName), true)
	viper.WriteConfig()
	return directory, nil
}

// Polupate a git host, such as github using a token auth with content of a folder.
// Use copy to flat the history
func PopulateRepoWithToken(owner string, repo string, sourceFolder string, gitHost string) error {

	//Clone Repo
	//Replace Content
	//Commit
	//Push

	config := configs.ReadConfig()
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Println("Unauthorized: No token present")
		return fmt.Errorf("missing github token")
	}
	directory := fmt.Sprintf("%s/push-%s", config.K1FolderPath, repo)
	err := os.RemoveAll(directory)
	if err != nil {
		log.Println("Error removing dir(expected if dir not present):", err)
	}
	url := fmt.Sprintf("https://%s@%s/%s/%s.git", token, gitHost, owner, repo)
	gitRepo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		log.Println("Error clonning git:", err)
		return err
	}

	w, _ := gitRepo.Worktree()
	log.Println("Committing new changes...")

	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			return false, nil

		},
	}
	err = cp.Copy(sourceFolder, directory, opt)
	if err != nil {
		log.Println("Error populating git")
		return err
	}
	w.Add(".")
	w.Commit("Populate Repo", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})

	err = gitRepo.Push(&git.PushOptions{
		RemoteName: "origin",
	})
	if err != nil {
		log.Println("error pushing to remote", err)
		return err
	}
	return nil
}

func CloneGitOpsRepo() {

	config := configs.ReadConfig()
	url := "https://github.com/kubefirst/gitops-template"
	directory := fmt.Sprintf("%s/gitops", config.K1FolderPath)

	versionGitOps := viper.GetString("gitops.branch")

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
			viper.Set(fmt.Sprintf("git.clone.%s.tag", repoName), fallbackTag)
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
