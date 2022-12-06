package pkg

import (
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/spf13/viper"
	"net/http"
)

// ForceLocalDestroy receives a GitHub client and use GitHub API to destroy GitHub recourses created during Kubefirst
// installation.
func ForceLocalDestroy(gitHubClient githubWrapper.GithubSession) error {

	owner := viper.GetString("github.owner")
	sshKeyId := viper.GetString("botpublickey")

	resp, err := gitHubClient.RemoveRepo(owner, "gitops")
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return err
	}
	resp, err = gitHubClient.RemoveRepo(owner, "metaphor")
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return err
	}
	resp, err = gitHubClient.RemoveRepo(owner, "metaphor-go")
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return err
	}
	resp, err = gitHubClient.RemoveRepo(owner, "metaphor-frontend")
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return err
	}

	err = gitHubClient.RemoveSSHKeyByPublicKey(owner, sshKeyId)
	if err != nil {
		return err
	}

	return nil
}
