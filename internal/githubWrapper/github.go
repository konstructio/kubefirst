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
