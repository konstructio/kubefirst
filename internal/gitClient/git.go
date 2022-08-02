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

func CloneRepoAndDetokenize(gitHost, githubOrg, repoName string, branch string) (string, error) {
	config := configs.ReadConfig()
	if branch == "" {
		branch = "main"
	}

	repoUrl := fmt.Sprintf("https://%s/%s/%s-template", gitHost, githubOrg, repoName)
	directory := fmt.Sprintf("%s/%s", config.K1FolderPath, repoName)
	log.Println("git clone", repoUrl, directory)
	log.Println("git clone -b ", branch, repoUrl, directory)

	_, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:           repoUrl,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		log.Println("error cloning %s-template repository from github %s", repoName, err)
		return directory, err
	}
	viper.Set(fmt.Sprintf("init.repos.%s.cloned", repoName), true)
	viper.WriteConfig()

	log.Printf("cloned %s-template repository to directory %s/%s", repoName, config.K1FolderPath, repoName)

	log.Printf("detokenizing %s/%s", config.K1FolderPath, repoName)
	pkg.Detokenize(directory)
	log.Printf("detokenization of %s/%s complete", config.K1FolderPath, repoName)

	viper.Set(fmt.Sprintf("init.repos.%s.detokenized", repoName), true)
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

	url := fmt.Sprintf("https://%s@%s/%s/%s.git", token, gitHost, owner, repo)
	gitRepo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		log.Println("Error clonning git")
		return err
	}

	w, _ := gitRepo.Worktree()
	log.Println("Committing new changes...")

	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			return strings.HasSuffix(src, ".git"), nil
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
