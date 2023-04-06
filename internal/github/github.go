/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

type GithubSession struct {
	context     context.Context
	staticToken oauth2.TokenSource
	oauthClient *http.Client
	gitClient   *github.Client
}

// New - Create a new client for github wrapper
func New(token string) GithubSession {
	if token == "" {
		log.Fatal().Msg("Unauthorized: No token present")
	}
	var gSession GithubSession
	gSession.context = context.Background()
	gSession.staticToken = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	gSession.oauthClient = oauth2.NewClient(gSession.context, gSession.staticToken)
	gSession.gitClient = github.NewClient(gSession.oauthClient)
	return gSession

}

func (g GithubSession) CreateWebhookRepo(org, repo, hookName, hookURL, hookSecret string, hookEvents []string) error {
	input := &github.Hook{
		Name:   &hookName,
		Events: hookEvents,
		Config: map[string]interface{}{
			"content_type": "json",
			"insecure_ssl": 0,
			"url":          hookURL,
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
		log.Fatal().Msg("No name: New repos must be given a name")
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

// RemoveRepo Removes a repository based on repository owner and name. It returns github.Response that hold http data,
// as http status code, the caller can make use of the http status code to validate the response.
func (g GithubSession) RemoveRepo(owner string, name string) (*github.Response, error) {
	if owner == "" {
		return nil, fmt.Errorf("a repository owner is required")
	}
	if name == "" {
		return nil, fmt.Errorf("a repository name is required")
	}

	resp, err := g.gitClient.Repositories.Delete(g.context, owner, name)
	if err != nil {
		return resp, fmt.Errorf("error removing private repo: %s - %s", name, err)
	}
	log.Printf("Successfully removed repo: %v\n", name)
	return resp, nil
}

// RemoveTeam - Remove  a team
func (g GithubSession) RemoveTeam(owner string, team string) error {
	if team == "" {
		log.Fatal().Msg("No name: repos must be given a name")
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
		log.Fatal().Msg("No name: repos must be given a name")
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

// RemoveSSHKeyByPublicKey deletes a GitHub key that matches the provided public key.
func (g GithubSession) RemoveSSHKeyByPublicKey(user string, publicKey string) error {

	keys, resp, err := g.gitClient.Users.ListKeys(g.context, user, &github.ListOptions{})
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to retrieve ssh keys, http code is: %d", resp.StatusCode)
	}

	for _, key := range keys {

		// as https://pkg.go.dev/golang.org/x/crypto/ssh@v0.0.0-20220722155217-630584e8d5aa#MarshalAuthorizedKey
		// documentation describes, the Marshall ssh key function adds extra new line at the end of the key id
		if key.GetKey()+"\n" == publicKey {
			resp, err := g.gitClient.Users.DeleteKey(g.context, key.GetID())
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusNoContent {
				return fmt.Errorf("unable to delete ssh-key, http code is: %d", resp.StatusCode)
			}
		}
	}

	return nil
}

// IsRepoInUse - Verify if a repo exists
func (g GithubSession) IsRepoInUse(org string, name string) (bool, error) {
	log.Printf("check if a repo is in use already")
	return false, nil
}

func (g GithubSession) CreatePR(
	branchName string,
	repoName string,
	gitHubUser string,
	baseBranch string,
	title string,
	body string) (*github.PullRequest, error) {

	head := branchName
	prData := github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Body:  &body,
		Base:  &baseBranch,
	}

	pullRequest, resp, err := g.gitClient.PullRequests.Create(
		context.Background(),
		gitHubUser,
		repoName,
		&prData,
	)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("pull request create response http code: %d", resp.StatusCode)

	return pullRequest, nil
}

func (g GithubSession) CommentPR(pullRequesrt *github.PullRequest, gitHubUser string, body string) error {

	issueComment := github.IssueComment{
		Body: &body,
	}

	_, resp, err := g.gitClient.Issues.CreateComment(
		context.Background(),
		gitHubUser,
		"gitops",
		*pullRequesrt.Number,
		&issueComment,
	)
	if err != nil {
		return err
	}
	log.Printf("pull request comment response http code: %d", resp.StatusCode)

	return nil

}

// SearchWordInPullRequestComment look for a specific sentence in a GitHub Pull Request comment
func (g GithubSession) SearchWordInPullRequestComment(gitHubUser string,
	gitOpsRepo string,
	pullRequest *github.PullRequest,
	searchFor string) (bool, error) {

	comments, r, err := g.gitClient.Issues.ListComments(
		context.Background(),
		gitHubUser,
		gitOpsRepo,
		*pullRequest.Number,
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

func (g GithubSession) RetrySearchPullRequestComment(
	gitHubUser string,
	gitOpsRepo string,
	pullRequest *github.PullRequest,
	searchFor string,
	logMessage string,
) (bool, error) {

	for i := 0; i < 30; i++ {
		ok, err := g.SearchWordInPullRequestComment(gitHubUser, gitOpsRepo, pullRequest, searchFor)
		if err != nil || !ok {
			log.Info().Msg(logMessage)
			time.Sleep(10 * time.Second)
			continue
		}
		return true, nil
	}
	return false, nil
}

// GetRepo - Always returns a status code for whether a repository exists or not
func (g GithubSession) CheckRepoExists(owner string, name string) int {
	_, response, _ := g.gitClient.Repositories.Get(g.context, owner, name)
	return response.StatusCode
}

// GetRepo - Always returns a status code for whether a team exists or not
func (g GithubSession) CheckTeamExists(owner string, name string) int {
	_, response, _ := g.gitClient.Teams.GetTeamBySlug(g.context, owner, name)
	return response.StatusCode
}

// DeleteRepositoryWebhook
func (g GithubSession) DeleteRepositoryWebhook(owner string, repository string, url string) error {
	webhooks, err := g.ListRepoWebhooks(owner, repository)
	if err != nil {
		return err
	}

	var hookID int64 = 0
	for _, hook := range webhooks {
		if url == hook.Config["url"] {
			hookID = hook.GetID()
		}
	}
	if hookID != 0 {
		_, err := g.gitClient.Repositories.DeleteHook(g.context, owner, repository, hookID)
		if err != nil {
			return err
		}
		log.Info().Msgf("deleted hook %s/%s/%s", owner, repository, url)
	} else {
		return fmt.Errorf("hook %s/%s/%s not found", owner, repository, url)
	}

	return nil
}

// ListRepoWebhooks returns all webhooks for a repository
func (g GithubSession) ListRepoWebhooks(owner string, repo string) ([]*github.Hook, error) {
	container := make([]*github.Hook, 0)
	for nextPage := 1; nextPage > 0; {
		hooks, resp, err := g.gitClient.Repositories.ListHooks(g.context, owner, repo, &github.ListOptions{
			Page:    nextPage,
			PerPage: 10,
		})
		if err != nil {
			return []*github.Hook{}, err
		}
		for _, hook := range hooks {
			container = append(container, hook)
		}
		nextPage = resp.NextPage
	}
	return container, nil
}
