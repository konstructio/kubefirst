package wrappers

import (
	"errors"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"log"
	"os"
)

// AuthenticateGitHubUserWrapper receives a handler that was previously instantiated, and communicate with GitHub.
// This wrapper is necessary to avoid code repetition when requesting GitHub PAT or Access token.
func AuthenticateGitHubUserWrapper(config *configs.Config, gitHubHandler *handlers.GitHubHandler) (string, error) {

	gitHubAccessToken := config.GitHubPersonalAccessToken
	if gitHubAccessToken != "" {
		return gitHubAccessToken, nil
	}

	gitHubAccessToken, err := gitHubHandler.AuthenticateUser()
	if err != nil {
		return "", err
	}

	if gitHubAccessToken == "" {
		return "", errors.New("unable to retrieve a GitHub token for the user")
	}

	if err := os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
		return "", err
	}
	log.Println("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")

	return gitHubAccessToken, nil
}
