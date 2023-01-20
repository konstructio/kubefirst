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
	"time"
)

// todo:
//  - add docs
//  - remove the hardcoded values
//  - you can complain with Joao if you see it after january 23
func TestGitHubUserCreation(t *testing.T) {

	//
	// git
	//
	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	if token == "" {
		t.Errorf("missing github token")
	}

	config := configs.ReadConfig()
	repoPath := config.K1FolderPath + "/gitops"
	repo, err := gitClient.CloneLocalRepo(repoPath)
	if err != nil {
		t.Errorf(err.Error())
	}

	workTree, err := gitClient.CheckoutBranch(repo, "main")
	if err != nil {
		t.Errorf(err.Error())
	}

	err = gitClient.PullBranch(workTree, "github", token)
	if err != nil {
		t.Errorf(err.Error())
	}

	err = gitClient.CreateBranch(repo, "test-branch6")
	if err != nil {
		t.Errorf(err.Error())
	}

	_, err = gitClient.CheckoutBranch(repo, "test-branch6")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Read the file contents into a variable
	adminGitHubFile := config.K1FolderPath + "/gitops/terraform/users/admins-github.tf"
	data, err := os.ReadFile(adminGitHubFile)
	if err != nil {
		t.Error()
	}

	var newFile []string
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		// remove line comment
		if strings.HasPrefix(line, "#") {
			line = strings.Replace(line, "#", "", 1)
		}

		// add extra "," to the end of the line for module concatenation
		if strings.Contains(line, "module.kubefirst_bot.vault_identity_entity_id") {
			line = line + ","
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
	// git
	//
	files := []string{"terraform/users/admins-github.tf"}

	err = gitClient.CommitFiles(workTree, "test commit 1", files)
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

	err = pkg.SetupViper(config)
	if err != nil {
		t.Error(err.Error())
	}
	gitHubUser := viper.GetString("github.user")

	gitHubClient := githubWrapper.New()
	err = gitHubClient.CreatePR(
		"test-branch6",
		"gitops",
		gitHubUser,
		"main",
		"testing123",
		"content body",
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	// todo: implement atlantis waiting
	// it will require to update atlantis waiting function, and will be done next
	println("waiting......")
	time.Sleep(60 * time.Second)
	//ok, err := gitHubClient.RetrySearchPullRequestComment(
	//	githubOwner,
	//	pkg.KubefirstGitOpsRepository,
	//	"To **apply** all unapplied plans from this pull request, comment",
	//	`waiting "atlantis plan" finish to proceed...`,
	//)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	err = gitHubClient.CommentPR(2, gitHubUser, "atlantis apply")
	if err != nil {
		t.Errorf(err.Error())
	}

}
