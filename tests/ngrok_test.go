package tests

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"net/http"
	"testing"
	"time"
)

// TestNgrokGitHubWebhookIntegration tests the ngrok GitHub webhook response, and look for a http response code of 200
func TestNgrokGitHubWebhookIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping end to tend test")
	}

	config := configs.ReadConfig()
	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err)
	}

	testCases := []struct {
		name     string
		url      string
		expected int
	}{
		{name: "ngrok", url: viper.GetString("ngrok.url") + "/events", expected: http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			client := &http.Client{
				Timeout: time.Second * 10,
			}
			resp, err := client.Get(tc.url)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			defer resp.Body.Close()

			fmt.Println("HTTP status code:", resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("HTTP status code is not 200")
			}
		})
	}

}
