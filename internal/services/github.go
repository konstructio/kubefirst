/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kubefirst/kubefirst/pkg"
)

type GitHubService struct {
	httpClient pkg.HTTPDoer
}

// gitHubAccessCode host OAuth data
type gitHubAccessCode struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// NewGitHubService instantiate a new GitHub service
func NewGitHubService(httpClient pkg.HTTPDoer) *GitHubService {
	return &GitHubService{
		httpClient: httpClient,
	}
}

// CheckUserCodeConfirmation checks if the user gave permission to the device flow request
func (service GitHubService) CheckUserCodeConfirmation(deviceCode string) (string, error) {

	gitHubAccessTokenURL := "https://github.com/login/oauth/access_token"

	jsonData, err := json.Marshal(map[string]string{
		"client_id":   pkg.GitHubOAuthClientId,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, gitHubAccessTokenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", pkg.JSONContentType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil
	}

	if res.StatusCode != http.StatusOK {
		log.Printf("waiting user to authorize at GitHub page..., current status code = %d", res.StatusCode)
		return "", fmt.Errorf("unable to issue a GitHub token")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", nil
	}

	var gitHubAccessToken gitHubAccessCode
	err = json.Unmarshal(body, &gitHubAccessToken)
	if err != nil {
		log.Println(err)
	}

	return gitHubAccessToken.AccessToken, nil
}
