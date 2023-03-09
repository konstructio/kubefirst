package metaphor_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kubefirst/kubefirst/configs"
)

// this is called when we want to make sure Metaphor are up and running
func TestMetaphorsLivenessIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()

	type conditions struct {
		serviceName    string
		serviceURL     string
		httpWantedCode int
	}

	testCases := []conditions{
		{
			serviceName:    "metaphor",
			serviceURL:     fmt.Sprintf("https://metaphor.%s", config.HostedZoneName),
			httpWantedCode: http.StatusOK,
		},
		{
			serviceName:    "metaphor-js",
			serviceURL:     fmt.Sprintf("https://metaphor-js.%s", config.HostedZoneName),
			httpWantedCode: http.StatusOK,
		},
		{
			serviceName:    "metaphor-go",
			serviceURL:     fmt.Sprintf("https://metaphor-go.%s", config.HostedZoneName),
			httpWantedCode: http.StatusOK,
		},
	}

	for _, wanted := range testCases {
		t.Run(wanted.serviceName, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(http.MethodGet, wanted.serviceURL, nil)
			if err != nil {
				t.Error(err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			if res.StatusCode != wanted.httpWantedCode {
				t.Errorf("wanted http status code 200, got %d", res.StatusCode)
			}
		})
	}

}
