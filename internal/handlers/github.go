package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
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

	requestBody, err := json.Marshal(map[string]string{
		"client_id": pkg.GitHubOAuthClientId,
		"scope":     "admin:org repo",
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

	var gitHubTokenReport bytes.Buffer
	gitHubTokenReport.WriteString(strings.Repeat("-", 69))
	gitHubTokenReport.WriteString("\nGitHub Token\n")
	gitHubTokenReport.WriteString(strings.Repeat("-", 69))
	gitHubTokenReport.WriteString("\n\nA GitHub token is required to provision a cluster using GitHub git provider. ")
	gitHubTokenReport.WriteString("Kubefirst can generate a token for your installation follow this steps: \n\n")
	//gitHubTokenReport.WriteString("1. click the following GitHub URL\n",)
	gitHubTokenReport.WriteString("1. click the following GitHub URL: " + gitHubDeviceFlow.VerificationUri + "\n")
	gitHubTokenReport.WriteString("2. entering the generate code: " + gitHubDeviceFlow.UserCode + "\n")
	gitHubTokenReport.WriteString("3. allow the organization")
	fmt.Println(reports.StyleMessage(gitHubTokenReport.String()))

	//fmt.Println("--- GitHub Auth Token data ---")
	//fmt.Printf("Copy the following code %s ,paste it at: %s , and authorize the organization \n", gitHubDeviceFlow.UserCode, gitHubDeviceFlow.VerificationUri)
	//fmt.Println("--- GitHub Auth Token data ---")

	var gitHubAccessToken string
	var attempts = 30
	for i := 0; i < attempts; i++ {
		gitHubAccessToken, err = handler.service.CheckUserCodeConfirmation(gitHubDeviceFlow.DeviceCode)
		if err != nil {
			log.Println(err)
		}

		if len(gitHubAccessToken) > 0 {
			fmt.Printf("\n\nGitHub token set!\n\n")
			return gitHubAccessToken, nil
		}
		fmt.Printf("\rwaiting for authorization (%d seconds)", (attempts)-i)
		// todo: handle github interval https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps#response-parameters
		time.Sleep(5 * time.Second)
	}
	return gitHubAccessToken, nil
}
