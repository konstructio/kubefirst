package githubWrapper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type GithubSession struct {
	context     context.Context
	staticToken oauth2.TokenSource
	oauthClient *http.Client
	gitClient   *github.Client
}

// New - Create a new client for github wrapper
func New() GithubSession {
	token := os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	var gSession GithubSession
	gSession.context = context.Background()
	gSession.staticToken = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	gSession.oauthClient = oauth2.NewClient(gSession.context, gSession.staticToken)
	gSession.gitClient = github.NewClient(gSession.oauthClient)
	return gSession

}

func (g GithubSession) CreateWebhookRepo(org, repo, hookName, hookUrl, hookSecret string, hookEvents []string) error {
	input := &github.Hook{
		Name:   &hookName,
		Events: hookEvents,
		Config: map[string]interface{}{
			"content_type": "json",
			"insecure_ssl": 0,
			"url":          hookUrl,
			"secret":       hookSecret,
		},
	}

	hook, _, err := g.gitClient.Repositories.CreateHook(g.context, org, repo, input)

	if err != nil {
		return fmt.Errorf("error when creating a webhook: %v", err)
	}

	log.Printf("Successfully created hook (id): %v", hook.GetID())

	return nil
}

// CreatePrivateRepo - Use github API to create a private repo
func (g GithubSession) CreatePrivateRepo(org string, name string, description string) error {
	if name == "" {
		log.Fatal("No name: New repos must be given a name")
	}
	isPrivate := true
	autoInit := true
	r := &github.Repository{Name: &name,
		Private:     &isPrivate,
		Description: &description,
		AutoInit:    &autoInit}
	repo, _, err := g.gitClient.Repositories.Create(g.context, org, r)
	if err != nil {
		return fmt.Errorf("error creating private repo: %s - %s", name, err)
	}
	log.Printf("Successfully created new repo: %v\n", repo.GetName())
	return nil
}

// RemoveRepo - Remove  a repo
func (g GithubSession) RemoveRepo(owner string, name string) error {
	if name == "" {
		log.Fatal("No name:  repos must be given a name")
	}
	_, err := g.gitClient.Repositories.Delete(g.context, owner, name)
	if err != nil {
		return fmt.Errorf("error removing private repo: %s - %s", name, err)
	}
	log.Printf("Successfully removed repo: %v\n", name)
	return nil
}

// RemoveTeam - Remove  a team
func (g GithubSession) RemoveTeam(owner string, team string) error {
	if team == "" {
		log.Fatal("No name:  repos must be given a name")
	}
	_, err := g.gitClient.Teams.DeleteTeamBySlug(g.context, owner, team)
	if err != nil {
		return fmt.Errorf("error removing team: %s - %s", team, err)
	}
	log.Printf("Successfully removed team: %v\n", team)
	return nil
}

// GetRepo - Returns  a repo
func (g GithubSession) GetRepo(owner string, name string) (*github.Repository, error) {
	if name == "" {
		log.Fatal("No name: repos must be given a name")
	}
	repo, _, err := g.gitClient.Repositories.Get(g.context, owner, name)
	if err != nil {
		return nil, fmt.Errorf("error removing private repo: %s - %s", name, err)
	}
	log.Printf("Successfully removed repo: %v\n", repo.GetName())
	return repo, nil
}

// AddSSHKey - Add ssh keys to a user account to allow kubefirst installer
// to use its own token during installation
func (g GithubSession) AddSSHKey(keyTitle string, publicKey string) (*github.Key, error) {
	log.Printf("Add SSH key to user account on behalf of kubefirst")
	key, _, err := g.gitClient.Users.CreateKey(g.context, &github.Key{Title: &keyTitle, Key: &publicKey})
	if err != nil {
		return nil, fmt.Errorf("error add SSH Key: %s", err)
	}
	return key, nil
}

// RemoveSSHKey - Removes SSH Key from github user
func (g GithubSession) RemoveSSHKey(keyId int64) error {
	log.Printf("Remove SSH key to user account on behalf of kubefrist")
	_, err := g.gitClient.Users.DeleteKey(g.context, keyId)
	if err != nil {
		return fmt.Errorf("error remiving SSH Key: %s", err)
	}
	return nil
}

// IsRepoInUse - Verify if a repo exists
func (g GithubSession) IsRepoInUse(org string, name string) (bool, error) {
	log.Printf("check if a repo is in use already")
	return false, nil
}

func (g GithubSession) CreatePR(branchName string) error {
	title := "update S3 backend to minio / internal k8s dns"
	head := branchName
	body := "use internal Kubernetes dns"
	base := "main"
	pr := github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Body:  &body,
		Base:  &base,
	}

	// todo: receive as parameter
	gitHubUser := viper.GetString("github.user")

	_, resp, err := g.gitClient.PullRequests.Create(
		context.Background(),
		gitHubUser,
		"gitops",
		&pr,
	)
	if err != nil {
		return err
	}
	log.Printf("pull request create response http code: %d", resp.StatusCode)

	return nil
}

func (g GithubSession) CommentPR(prNumber int, body string) error {

	issueComment := github.IssueComment{
		Body: &body,
	}

	// todo: receive as parameter
	gitHubUser := viper.GetString("github.user")

	_, resp, err := g.gitClient.Issues.CreateComment(
		context.Background(),
		gitHubUser,
		"gitops", prNumber,
		&issueComment,
	)
	if err != nil {
		return err
	}
	log.Printf("pull request comment response http code: %d", resp.StatusCode)

	return nil

}

// SearchWordInPullRequestComment look for a specific sentence in a GitHub Pull Request comment
func (g GithubSession) SearchWordInPullRequestComment(gitHubUser string, gitOpsRepo string, searchFor string) (bool, error) {

	comments, r, err := g.gitClient.Issues.ListComments(
		context.Background(),
		gitHubUser,
		gitOpsRepo,
		1,
		&github.IssueListCommentsOptions{},
	)
	if err != nil {
		return false, err
	}

	if r.StatusCode != http.StatusOK {
		return false, nil
	}

	for _, v := range comments {
		if strings.Contains(*v.Body, searchFor) {
			return true, nil
		}
	}

	return false, nil
}

// todo: not sure if this is the right place for this function
func (g GithubSession) RetrySearchPullRequestComment(
	gitHubUser string,
	gitOpsRepo string,
	searchFor string,
	logMessage string,
) (bool, error) {

	for i := 0; i < 30; i++ {
		ok, err := g.SearchWordInPullRequestComment(gitHubUser, gitOpsRepo, searchFor)
		if err != nil || !ok {
			log.Println(logMessage)
			time.Sleep(10 * time.Second)
			continue
		}
		return true, nil
	}
	return false, nil
}
