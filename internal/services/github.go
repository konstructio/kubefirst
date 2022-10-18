package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type GitHubService struct{}

type TokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func NewGitHubService() *GitHubService {
	return &GitHubService{}
}

func (service GitHubService) PoolAccessToken(clientId string, deviceCode string) string {
	urlA := "https://github.com/login/oauth/access_token"

	//payload := strings.NewReader("client_id=cfe20fec21fd8126d9be&device_code=f479009c42646ea8a5424ffb8ff0c6884ead9575&grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code")
	payload := url.Values{}
	payload.Add("client_id", clientId)
	payload.Add("device_code", deviceCode)
	grantType := "urn:ietf:params:oauth:grant-type:device_code"
	payload.Add("grant_type", grantType)

	req, _ := http.NewRequest("POST", urlA, strings.NewReader(payload.Encode()))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var tk TokenResp
	err := json.Unmarshal(body, &tk)
	if err != nil {
		log.Println(err)
	}

	fmt.Println("---debug1---")
	fmt.Println(tk.AccessToken)
	fmt.Println("---debug1---")

	return "a"
}
