package tests

import (
	"errors"
	"github.com/go-git/go-git/v5"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"os"
	"strings"
	"testing"
)

// TestGitHubUserCreationEndToEnd is an end-to-end test that creates a new branch, modifies a file, commits the changes,
// pushes to remote and creates a pull request on GitHub. It requires KUBEFIRST_GITHUB_AUTH_TOKEN environment variable
// to be set in order to push the changes and create the pull request on GitHub. Unit tests will ignore it.
func TestGitHubUserCreationEndToEnd(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping end to tend test")
	}

	//
	// setup
	//
	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	if token == "" {
		t.Errorf("missing github token")
	}

	config := configs.ReadConfig()
	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err.Error())
	}

	baseBranch := "main"
	branchName := "e2e_add_new_user"
	repoPath := config.K1FolderPath + "/gitops"
	repo, err := gitClient.CloneLocalRepo(repoPath)
	if err != nil {
		t.Errorf(err.Error())
	}

	//
	// prepare git
	//
	workTree, err := gitClient.CheckoutBranch(repo, baseBranch)
	if err != nil {
		t.Errorf(err.Error())
	}

	err = gitClient.PullBranch(workTree, "github", token)
	if err != nil {
		t.Errorf(err.Error())
	}

	err = gitClient.CreateBranch(repo, branchName)
	if err != nil {
		t.Errorf(err.Error())
	}

	_, err = gitClient.CheckoutBranch(repo, branchName)
	if err != nil {
		t.Errorf(err.Error())
	}

	//
	// file manipulation
	//

	// Read the file contents into a variable
	adminGitHubFile := config.K1FolderPath + "/gitops/terraform/users/admins-github.tf"
	data, err := os.ReadFile(adminGitHubFile)
	if err != nil {
		t.Error()
	}

	var newFile []string
	lines := strings.Split(string(data), "\n")

	// remove commented lines
	for _, line := range lines {
		// remove line comment
		if strings.HasPrefix(line, "#") {
			line = strings.Replace(line, "#", "", 1)
		}

		// add extra "," to the end of the line for module concatenation
		if strings.Contains(line, "module.kubefirst_bot.vault_identity_entity_id") {
			line = line + ","
		}
		if strings.Contains(line, "admin_one_github_username") {
			line = strings.Replace(line, "admin_one_github_username", "adminone_gh_user", 1)
		}

		newFile = append(newFile, line+"\n")
	}

	f, err := os.Create(adminGitHubFile)
	if err != nil {
		t.Errorf(err.Error())
	}

	for _, line := range newFile {
		_, err = f.WriteString(line)
		if err != nil {
			t.Errorf(err.Error())
		}
	}

	//
	// update git remote
	//
	files := []string{"terraform/users/admins-github.tf"}

	err = gitClient.CommitFiles(workTree, "[e2e] add new user", files)
	if err != nil {
		t.Errorf(err.Error())
	}

	err = gitClient.PushChanges(repo, "github", token)
	if errors.Is(err, git.ErrNonFastForwardUpdate) {
		println("non fast forward update")
		return
	}
	if err != nil {
		t.Errorf(err.Error())
	}

	gitHubOwner := viper.GetString("github.owner")

	gitHubClient := githubWrapper.New()
	pullRequest, err := gitHubClient.CreatePR(
		branchName,
		viper.GetString("gitops.repo"),
		gitHubOwner,
		baseBranch,
		"[e2e] add new user",
		"this is automatically created by Kubefirst e2e test",
	)
	if err != nil {
		t.Errorf(err.Error())
	}

	// wait for atlantis update
	ok, err := gitHubClient.RetrySearchPullRequestComment(
		gitHubOwner,
		pkg.KubefirstGitOpsRepository,
		pullRequest,
		"To **apply** all unapplied plans from this pull request, comment",
		`waiting "atlantis plan" finish to proceed...`,
	)
	if ok != true {
		t.Errorf("atlantis plan failed")
	}
	if err != nil {
		t.Error(err.Error())
	}
	err = gitHubClient.CommentPR(pullRequest, gitHubOwner, "atlantis apply")
	if err != nil {
		t.Errorf(err.Error())
	}

}
