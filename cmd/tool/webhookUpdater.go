package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//Run a constant checket for ngrok hook state and update  client if found changes.

func runWebhookUpdater(cmd *cobra.Command, args []string) error {
	//while true
	//Call Some SDK to validate NGROK
	//CURL: curl http://localhost:3000/api/tunnels
	//Check if previous tunnel is the same of the new tunnel public address
	//if matches, do nothing - sleep/exit
	//if don't matches
	//USE GITHUB token from ENV Variable(it will come from vault later)
	// USe token to create a webhook
	// USe token to remove  old webhook
	//Where old webhook is stored?
	//	- configMap?
	//	- can it be queried by some label on github?
	// - start witha pre-defined name
	gitHubClient := githubWrapper.New()
	clientset, err := k8s.GetClientSet(false)
	if err != nil {
		return fmt.Errorf("error when connecting to k8s: %v", err)
	}
	atlantisSecretClient := clientset.CoreV1().Secrets("atlantis")
	for true {
		payload, err := CheckNgrokTunnel()
		if err != nil {
			log.Warn().Msgf("error checking status of tunnel: %v", err)
			// We will try again soon, once cluster is more ready
			continue
		}
		if len(payload.Tunnels) > 0 {
			log.Warn().Msgf("error reading tunnel info:  no tunnels")
			// We will try again soon, once cluster is more ready
			continue
		}

		hookName := "ngrok_atlantis"
		hookURL := payload.Tunnels[0].PublicURL + "/events"
		if hookURL == lastTunnel {
			// Nothing to be done
			continue
		}
		hookSecret := k8s.GetSecretValue(atlantisSecretClient, "atlantis-secrets", "ATLANTIS_GH_WEBHOOK_SECRET")
		hookEvents := []string{"issue_comment", "pull_request", "pull_request_review", "push"}
		err = gitHubClient.UpdateWebhook(owner, repo, hookName, hookURL, hookSecret, hookEvents)
		if err != nil {
			return fmt.Errorf("error when updating a webhook: %v", err)
		}
		lastTunnel = hookURL
		time.Sleep(20 * time.Second)
	}

	return nil
}

//atlantis  get secrets atlantis-secrets

func validateWebhookUpdater(cmd *cobra.Command, args []string) error {
	if len(repo) < 1 || len(owner) < 1 {
		return fmt.Errorf("both repo(%s) and owner(%s) must be provided in order for webhookupdater to work as expected", repo, owner)
	}
	log.Info().Msgf("Validation: Success repo(%s) and owner(%s) provided as epxected", repo, owner)
	return nil
}

type NgrokTunnel struct {
	Tunnels []struct {
		Name      string `json:"name"`
		ID        string `json:"ID"`
		URI       string `json:"uri"`
		PublicURL string `json:"public_url"`
		Proto     string `json:"proto"`
		Config    struct {
			Addr    string `json:"addr"`
			Inspect bool   `json:"inspect"`
		} `json:"config"`
	} `json:"tunnels"`
	URI string `json:"uri"`
}

func CheckNgrokTunnel() (*NgrokTunnel, error) {

	url := "http://ngrok.ngrok-agent.svc.cluster.local:4040/api/tunnels"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return &NgrokTunnel{}, err
	}
	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", pkg.JSONContentType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return &NgrokTunnel{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return &NgrokTunnel{}, err
	}

	var payload NgrokTunnel
	err = json.Unmarshal(body, &payload)
	if err != nil {
		log.Warn().Msgf("%s", err)
	}
	return &payload, nil
}
