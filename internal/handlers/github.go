package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// GitHubDeviceFlow handles https://docs.github.com/apps/building-oauth-apps/authorizing-oauth-apps#device-flow
type GitHubDeviceFlow struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUri string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// GitHubHandler receives a GitHubService
type GitHubHandler struct {
	service *services.GitHubService
}

// NewGitHubHandler instantiate a new GitHub handler
func NewGitHubHandler(gitHubService *services.GitHubService) *GitHubHandler {
	return &GitHubHandler{
		service: gitHubService,
	}
}

// AuthenticateUser initiate the GitHub Device Login Flow. First step is to issue a new device, and user code. Next it
// waits for the user authorize the request in the browser, then it pool GitHub access point endpoint, to validate and
// grant permission to return a valid access token.
func (handler GitHubHandler) AuthenticateUser() (string, error) {

	gitHubDeviceFlowCodeURL := "https://github.com/login/device/code"
	// todo: update scope list, we have more than we need at the moment
	requestBody, err := json.Marshal(map[string]string{
		"client_id": pkg.GitHubOAuthClientId,
		"scope":     "repo public_repo admin:repo_hook admin:org admin:public_key admin:org_hook user project delete_repo write:packages admin:gpg_key workflow",
	})

	req, err := http.NewRequest(http.MethodPost, gitHubDeviceFlowCodeURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", pkg.JSONContentType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var gitHubDeviceFlow GitHubDeviceFlow
	err = json.Unmarshal(body, &gitHubDeviceFlow)
	if err != nil {
		log.Println(err)
	}

	// todo: check http code

	// UI update to the user adding instructions how to proceed
	gitHubTokenReport := reports.GitHubAuthToken(gitHubDeviceFlow.UserCode, gitHubDeviceFlow.VerificationUri)
	fmt.Println(reports.StyleMessage(gitHubTokenReport))

	// todo add a 10 second countdown to warn browser open
	time.Sleep(8 * time.Second)
	exec.Command("open", "https://github.com/login/device").Start()

	// todo: improve the logic for the counter
	var gitHubAccessToken string
	var attempts = 10
	var attemptsControl = attempts + 90
	for i := 0; i < attempts; i++ {
		gitHubAccessToken, err = handler.service.CheckUserCodeConfirmation(gitHubDeviceFlow.DeviceCode)
		if err != nil {
			log.Println(err)
		}

		if len(gitHubAccessToken) > 0 {
			githubOwner := getGithubOwner(gitHubAccessToken)

			fmt.Printf("\n\nGitHub token set!\n\n")
			viper.Set("github.token", gitHubAccessToken)
			viper.Set("github.user", githubOwner) // TODO: deal with it
			viper.Set("github.owner", githubOwner)
			viper.WriteConfig()
			return gitHubAccessToken, nil
		}
		fmt.Printf("\rwaiting for authorization (%d seconds)", (attemptsControl)-5)
		attemptsControl -= 5
		// todo: handle github interval https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps#response-parameters
		time.Sleep(5 * time.Second)
	}
	return gitHubAccessToken, nil
}

func getGithubOwner(gitHubAccessToken string) string {

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		log.Println("error setting request")
	}
	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gitHubAccessToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("error doing request")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("error unmarshalling request")
	}

	type GitHubUser struct {
		Login string `json:"login"`
	}

	var githubUser GitHubUser
	err = json.Unmarshal(body, &githubUser)
	if err != nil {
		log.Println(err)
	}
	log.Println(githubUser.Login)
	return githubUser.Login

}
