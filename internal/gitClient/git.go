package gitClient

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	cp "github.com/otiai10/copy"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

// Github - git-provider github
const Github = "github"

// Gitlab - git-provider github
const Gitlab = "gitlab"

// CloneRepoAndDetokenizeTemplate - clone repo using CloneRepoAndDetokenizeTemplate that uses fallback rule to try to capture version
func CloneRepoAndDetokenizeTemplate(githubOwner, repoName, folderName string, branch string, tag string) (string, error) {
	config := configs.ReadConfig()

	directory := fmt.Sprintf("%s/%s", config.K1FolderPath, folderName)
	err := os.RemoveAll(directory)
	if err != nil {
		log.Error().Err(err).Msg("Error removing dir(expected if dir not present):")
	}

	err = CloneTemplateRepoWithFallBack(githubOwner, repoName, directory, branch, tag)
	if err != nil {
		log.Panic().Err(err).Msg("Error cloning repo with fallback")
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
	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Info().Msg("Unauthorized: No token present")
		return fmt.Errorf("missing github token")
	}
	directory := fmt.Sprintf("%s/push-%s", config.K1FolderPath, repo)
	err := os.RemoveAll(directory)
	if err != nil {
		log.Error().Err(err).Msg("Error removing dir(expected if dir not present)")
	}
	url := fmt.Sprintf("https://%s@%s/%s/%s.git", token, gitHost, owner, repo)
	gitRepo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		log.Error().Err(err).Msg("Error clonning git")
		return err
	}

	w, _ := gitRepo.Worktree()
	log.Info().Msg("Committing new changes...")

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
		log.Info().Msg("Error populating git")
		return err
	}
	status, err := w.Status()
	if err != nil {
		log.Error().Err(err).Msg("error getting worktree status")
	}

	for file, s := range status {
		log.Printf("the file is %s the status is %v", file, s.Worktree)
		_, err = w.Add(file)
		if err != nil {
			log.Error().Err(err).Msg("error getting worktree status")
		}
	}
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
		log.Error().Err(err).Msg("error pushing to remote")
		return err
	}
	return nil
}

func CloneGitOpsRepo() {

	config := configs.ReadConfig()
	url := "https://github.com/kubefirst/gitops-template"
	directory := fmt.Sprintf("%s/gitops", config.K1FolderPath)

	versionGitOps := viper.GetString("gitops.branch")

	log.Info().Msgf("git clone -b %s %s %s", versionGitOps, url, directory)

	_, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(versionGitOps),
		SingleBranch:  true,
	})
	if err != nil {
		log.Panic().Err(err).Msg("error cloning gitops-template repository from github")
	}

	log.Info().Msgf("downloaded gitops repo from template to directory %s%s", config.K1FolderPath, "/gitops")
}

func PushGitopsToSoftServe() {
	cfg := configs.ReadConfig()
	directory := fmt.Sprintf("%s/gitops", cfg.K1FolderPath)

	log.Info().Msgf("open gitClient repo %s", directory)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Panic().Err(err).Msgf("error opening the directory %q", directory)
	}

	log.Info().Msg("gitClient remote add origin ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops")
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "soft",
		URLs: []string{"ssh://127.0.0.1:8022/gitops"},
	})
	if err != nil {
		log.Panic().Err(err).Msgf("Error creating remote repo")
	}
	w, _ := repo.Worktree()

	log.Info().Msg("Committing new changes...")
	status, err := w.Status()
	if err != nil {
		log.Error().Err(err).Msg("error getting worktree status")
	}

	for file, s := range status {
		log.Printf("the file is %s the status is %v", file, s.Worktree)
		_, err = w.Add(file)
		if err != nil {
			log.Error().Err(err).Msg("error getting worktree status")
		}
	}
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
		log.Error().Err(err).Msg("error pushing to remote")
	}

}

// CloneTemplateRepoWithFallBack - Tries to clone branch, if defined, else try to clone Tag
// In the absence of matching tag/branch function will fail
func CloneTemplateRepoWithFallBack(githubOrg string, repoName string, directory string, branch string, fallbackTag string) error {
	defer viper.WriteConfig()
	// todo need to refactor this and have the repoName include -template
	githubOrg = "kubefirst"
	repoURL := fmt.Sprintf("https://github.com/%s/%s-template", githubOrg, repoName)

	isMainBranch := true
	isRepoClone := false
	source := ""
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
			source = "branch"
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
			source = "tag"
			viper.Set(fmt.Sprintf("git.clone.%s.tag", repoName), fallbackTag)
		}
	}

	if !isRepoClone {
		log.Printf("Error cloning template of repos, code not found on Branch(%s) or Tag(%s) of repo: %s", branch, fallbackTag, repoURL)
		return fmt.Errorf("error cloning template, No templates found on branch or tag")
	}

	w, _ := repo.Worktree()
	if !isMainBranch {
		branchName := plumbing.NewBranchReferenceName("main")
		headRef, err := repo.Head()
		if err != nil {
			log.Panic().Err(err).Msgf("Error Setting reference: %s", repoName)
		}
		ref := plumbing.NewHashReference(branchName, headRef.Hash())
		err = repo.Storer.SetReference(ref)
		if err != nil {
			log.Panic().Err(err).Msgf("error Storing reference: %s", repoName)
		}
		err = w.Checkout(&git.CheckoutOptions{Branch: ref.Name()})
		//remove old branch
		err = repo.Storer.RemoveReference(plumbing.NewBranchReferenceName(branch))
		if err != nil {
			if source == "branch" {
				log.Panic().Err(err).Msgf("error removing old branch: %s", repoName)
			} else {
				//this code will probably fail from a tag sourced clone
				//post-1.9.0 tag some tests will be done to ensure the final logic.
				log.Printf("[101] error removing old branch: %s, %s", repoName, err)
			}
		}

	}
	return nil

}

func PushLocalRepoToEmptyRemote(githubHost, githubOwner, localRepo, remoteName string) {
	cfg := configs.ReadConfig()

	localDirectory := fmt.Sprintf("%s/%s", cfg.K1FolderPath, localRepo)

	log.Info().Msgf("opening repository with gitClient: %q", localDirectory)
	repo, err := git.PlainOpen(localDirectory)
	if err != nil {
		log.Panic().Err(err).Msgf("error opening the localDirectory: %s", localDirectory)
	}

	w, _ := repo.Worktree()

	log.Info().Msg("Committing new changes... PushLocalRepoToEmptyRemote")

	status, err := w.Status()
	if err != nil {
		log.Error().Err(err).Msg("error getting worktree status")
	}

	for file, _ := range status {
		_, err = w.Add(file)
		if err != nil {
			log.Error().Err(err).Msg("error getting worktree status")
		}
	}
	w.Commit("setting new remote upstream to github", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})

	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	if len(token) == 0 {
		log.Info().Msg("no GITHUB KUBEFIRST_GITHUB_AUTH_TOKEN provided, unable to use GitHub API")
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		Auth: &http.BasicAuth{
			Username: "kubefirst-bot",
			Password: token,
		},
	})
	if err != nil {
		log.Panic().Err(err).Msgf("error pushing to remote %s", remoteName)
	}
	log.Info().Msgf("successfully pushed detokenized gitops content to github/%s", viper.GetString("github.owner"))
	viper.Set("github.gitops.hydrated", true)
	viper.WriteConfig()
}

func PushLocalRepoUpdates(githubHost, githubOwner, localRepo, remoteName string) {

	cfg := configs.ReadConfig()

	localDirectory := fmt.Sprintf("%s/%s", cfg.K1FolderPath, localRepo)
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/vault/.terraform", cfg.K1FolderPath))
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/vault/.terraform.lock.hcl", cfg.K1FolderPath))

	log.Info().Msgf("opening repository with gitClient: %s", localDirectory)
	repo, err := git.PlainOpen(localDirectory)
	if err != nil {
		log.Panic().Err(err).Msgf("error opening the localDirectory: %s", localDirectory)
	}

	url := fmt.Sprintf("https://%s/%s/%s", githubHost, githubOwner, localRepo)
	log.Printf("git push to  remote: %s url: %s", remoteName, url)

	w, _ := repo.Worktree()

	log.Info().Msg("Committing new changes... PushLocalRepoUpdates")
	status, err := w.Status()
	if err != nil {
		log.Error().Err(err).Msg("error getting worktree status")
	}

	for file, s := range status {
		log.Printf("the file is %s the status is %v", file, s.Worktree)
		_, err = w.Add(file)
		if err != nil {
			log.Error().Err(err).Msg("error getting worktree status")
		}
	}
	w.Commit("commiting staged changes to remote", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})

	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	err = repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		Auth: &http.BasicAuth{
			Username: "kubefirst-bot",
			Password: token,
		},
	})
	if err != nil {
		log.Panic().Err(err).Msg("error pushing to remote")
	}
	log.Info().Msgf("successfully pushed detokenized gitops content to github/%s", viper.GetString("github.owner"))
}

// todo: refactor
func UpdateLocalTerraformFilesAndPush(githubHost, githubOwner, localRepo, remoteName string, branchDestiny plumbing.ReferenceName) error {

	cfg := configs.ReadConfig()

	localDirectory := fmt.Sprintf("%s/%s", cfg.K1FolderPath, localRepo)
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/vault/.terraform", cfg.K1FolderPath))
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/vault/.terraform.lock.hcl", cfg.K1FolderPath))
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/github/.terraform", cfg.K1FolderPath))
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/github/.terraform.lock.hcl", cfg.K1FolderPath))
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/github/terraform.tfstate", cfg.K1FolderPath))
	os.RemoveAll(fmt.Sprintf("%s/gitops/terraform/github/terraform.tfstate.backup", cfg.K1FolderPath))

	log.Info().Msgf("opening repository with gitClient: %s", localDirectory)
	repo, err := git.PlainOpen(localDirectory)
	if err != nil {
		log.Panic().Err(err).Msgf("error opening the localDirectory: %s", localDirectory)
	}

	url := fmt.Sprintf("https://%s/%s/%s", githubHost, githubOwner, localRepo)
	log.Printf("git push to  remote: %s url: %s", remoteName, url)

	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: branchDestiny,
		Create: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	log.Info().Msg("Committing new changes... PushLocalRepoUpdates")

	if viper.GetString("gitprovider") == "github" {
		gitHubRemoteBackendFiled := "terraform/users/kubefirst-github.tf"
		_, err = w.Add(gitHubRemoteBackendFiled)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		remoteBackendFile := "terraform/github/remote-backend.tf"
		_, err = w.Add(remoteBackendFile)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
	}
	vaultMainFile := "terraform/vault/main.tf"
	_, err = w.Add(vaultMainFile)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	_, err = w.Commit("update s3 terraform backend to minio", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	err = repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		Auth: &http.BasicAuth{
			Username: "kubefirst-bot",
			Password: token,
		},
	})
	if err != nil {
		log.Panic().Err(err).Msg("error pushing to remote")
	}
	log.Info().Msgf("successfully pushed detokenized gitops content to github/%s", viper.GetString("github.owner"))

	return nil
}

// CloneBranch clone a branch and returns a pointer to git.Repository
func CloneBranch(repoURL string, repoLocalPath string, branch string) (*git.Repository, error) {

	log.Printf("git cloning by branch, branch: %s", configs.K1Version)

	repo, err := git.PlainClone(repoLocalPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// CloneBranchSetMain clone a branch and returns a pointer to git.Repository
func CloneBranchSetMain(repoURL string, repoLocalPath string, branch string) (*git.Repository, error) {

	log.Printf("git cloning by branch, branch: %s", configs.K1Version)

	repo, err := CloneBranch(repoURL, repoLocalPath, branch)
	if err != nil {
		return nil, err
	}
	if branch != "main" {
		repo, err = SetToMainBranch(repo)
		if err != nil {
			return nil, fmt.Errorf("error setting repository main from GitHub using branch %s", branch)
		}
		//remove old branch
		err = repo.Storer.RemoveReference(plumbing.NewBranchReferenceName(branch))
		if err != nil {
			return nil, fmt.Errorf("error removing previous branch: %s", err)
		}
	}
	return repo, nil
}

// CheckoutBranch checkout a branch
func CheckoutBranch(repo *git.Repository, branch string) error {

	tree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = tree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return err
	}

	return nil
}

// CloneTag clone a repository using a tag value, and returns a pointer to *git.Repository
func CloneTag(repoLocalPath string, githubOrg string, repoName string, tag string) (*git.Repository, error) {

	// todo: repoURL como param
	repoURL := fmt.Sprintf("https://github.com/%s/%s-template", githubOrg, repoName)

	log.Printf("git cloning by tag, tag: %s", configs.K1Version)

	repo, err := git.PlainClone(repoLocalPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewTagReferenceName(tag),
		SingleBranch:  true,
	})
	if err != nil {
		log.Printf("error cloning %s-template repository from GitHub using tag %s", repoName, configs.K1Version)
		return nil, err
	}

	return repo, nil
}

// CloneTagSetMain  CloneTag plus fixes branch to be main
func CloneTagSetMain(repoLocalPath string, githubOrg string, repoName string, tag string) (*git.Repository, error) {

	repo, err := CloneTag(repoLocalPath, githubOrg, repoName, tag)
	if err != nil {
		log.Printf("error cloning %s-template repository from GitHub using tag %s", repoName, configs.K1Version)
		return nil, err
	}
	repo, err = SetToMainBranch(repo)
	if err != nil {
		log.Printf("error setting main for %s  repository from GitHub using tag %s", repoName, configs.K1Version)
		return nil, err
	}

	return repo, nil
}

// SetToMainBranch point branch or tag to main
func SetToMainBranch(repo *git.Repository) (*git.Repository, error) {
	w, _ := repo.Worktree()
	branchName := plumbing.NewBranchReferenceName("main")
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("Error Setting reference: %s", err)
	}

	ref := plumbing.NewHashReference(branchName, headRef.Hash())
	err = repo.Storer.SetReference(ref)
	if err != nil {
		return nil, fmt.Errorf("error Storing reference: %s", err)
	}

	err = w.Checkout(&git.CheckoutOptions{Branch: ref.Name()})
	if err != nil {
		return nil, fmt.Errorf("error checking out main: %s", err)
	}
	return repo, nil
}

// CheckoutTag repository checkout based on a tag
func CheckoutTag(repo *git.Repository, tag string) error {

	tree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = tree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/tags/" + tag),
	})
	if err != nil {
		return err
	}

	return nil
}

// CreateGitHubRemote create a remote repository entry
func CreateGitHubRemote(gitOpsLocalRepoPath string, gitHubUser string, repoName string) error {

	log.Info().Msg("creating git remote (github)...")

	repo, err := git.PlainOpen(gitOpsLocalRepoPath)
	if err != nil {
		return err
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s", gitHubUser, repoName)

	_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "github",
		URLs: []string{repoURL},
	})
	if err != nil {
		return err
	}

	log.Info().Msg("creating git remote (github) done")

	return nil
}
