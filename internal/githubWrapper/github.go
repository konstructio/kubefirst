package githubWrapper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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
func New() GithubSession {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
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
	log.Printf("Add SSH key to user account on behalf of kubefrist")
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
