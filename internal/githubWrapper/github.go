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

type githubSession struct {
	context     context.Context
	staticToken oauth2.TokenSource
	oauthClient *http.Client
	gitClient   *github.Client
}

func New() githubSession {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	var gSession githubSession
	gSession.context = context.Background()
	gSession.staticToken = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	gSession.oauthClient = oauth2.NewClient(gSession.context, gSession.staticToken)
	gSession.gitClient = github.NewClient(gSession.oauthClient)
	return gSession

}

func (g githubSession) createPrivateRepo(org string, name string, description string) {
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
		log.Fatal(err)
	}
	fmt.Printf("Successfully created new repo: %v\n", repo.GetName())
}
