package metaphor_test

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"net/http"
	"testing"
)

// this is called when we want to make sure Metaphors are up and running
func TestMetaphorsLivenessIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()
	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err)
	}

	type conditions struct {
		serviceName    string
		serviceURL     string
		httpWantedCode int
	}

	testCases := []conditions{
		{
			serviceName:    "metaphor-frontend",
			serviceURL:     fmt.Sprintf("https://metaphor-frontend.%s", viper.GetString("aws.hostedzonename")),
			httpWantedCode: http.StatusOK,
		},
		{
			serviceName:    "metaphor-js",
			serviceURL:     fmt.Sprintf("https://metaphor-js.%s", viper.GetString("aws.hostedzonename")),
			httpWantedCode: http.StatusOK,
		},
		{
			serviceName:    "metaphor-go",
			serviceURL:     fmt.Sprintf("https://metaphor-go.%s", viper.GetString("aws.hostedzonename")),
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
