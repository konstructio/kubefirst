package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
)

// GitHubDeviceFlow handles https://docs.github.com/apps/building-oauth-apps/authorizing-oauth-apps#device-flow
type GitHubDeviceFlow struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUri string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type GitHubUser struct {
	Login string `json:"login"`
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

	gitHubDeviceFlowCodeURL := pkg.GitHubLoginDeviceURL + "/code"
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
		log.Warn().Msgf("%s", err)
	}

	// todo: check http code

	// UI update to the user adding instructions how to proceed
	gitHubTokenReport := reports.GitHubAuthToken(gitHubDeviceFlow.UserCode, gitHubDeviceFlow.VerificationUri)
	fmt.Println(reports.StyleMessage(gitHubTokenReport))

	fmt.Println(reports.StyleMessage("Please press <enter> to open the GitHub page:"))
	// this blocks the progress until the user hits enter to open the browser
	if _, err = fmt.Scanln(); err != nil {
		return "", err
	}

	if err = pkg.OpenBrowser(pkg.GitHubLoginDeviceURL); err != nil {
		return "", err
	}

	var gitHubAccessToken string
	var attempts = 18       // 18 * 5 = 90 seconds
	var secondsControl = 95 // 95 to start with 95-5=90
	for i := 0; i < attempts; i++ {
		gitHubAccessToken, err = handler.service.CheckUserCodeConfirmation(gitHubDeviceFlow.DeviceCode)
		if err != nil {
			log.Warn().Msgf("%s", err)
		}

		if len(gitHubAccessToken) > 0 {
			fmt.Printf("\n\nGitHub token set!\n\n")
			return gitHubAccessToken, nil
		}

		secondsControl -= 5
		fmt.Printf("\rwaiting for authorization (%d seconds)", secondsControl)
		// todo: handle github interval https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps#response-parameters
		time.Sleep(5 * time.Second)
	}
	fmt.Println("") // will avoid writing the next print in the same line
	return gitHubAccessToken, nil
}

// todo: make it a method
func (handler GitHubHandler) GetGitHubUser(gitHubAccessToken string) (string, error) {

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		log.Warn().Msg("error setting request")
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gitHubAccessToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"something went wrong calling GitHub API, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	var githubUser GitHubUser
	err = json.Unmarshal(body, &githubUser)
	if err != nil {
		return "", err
	}

	if len(githubUser.Login) == 0 {
		return "", errors.New("unable to retrieve username via GitHub API")
	}

	log.Info().Msgf("GitHub user: %s", githubUser.Login)
	return githubUser.Login, nil

}
