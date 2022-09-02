package gitlab_test

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"net/http"
	"testing"
)

// this is called when GitLab should be up and running
func TestGitLabLivenessIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()
	if len(config.HostedZoneName) == 0 {
		t.Error("HOSTED_ZONE_NAME environment variable is not set")
		return
	}

	argoURL := fmt.Sprintf("https://gitlab.%s", config.HostedZoneName)

	req, err := http.NewRequest(http.MethodGet, argoURL, nil)
	if err != nil {
		t.Error(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("wanted http status code 200, got %d", res.StatusCode)
	}
}
