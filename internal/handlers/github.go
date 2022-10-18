package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Device1 struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUri string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type GitHubHandler struct {
	service *services.GitHubService
}

func NewGitHubHandler(gitHubService *services.GitHubService) *GitHubHandler {
	return &GitHubHandler{
		service: gitHubService,
	}
}

func (handler GitHubHandler) AuthenticateUser() error {

	url1 := "https://github.com/login/device/code"

	payload := url.Values{}
	payload.Add("client_id", pkg.GitHubOAuthClientId)
	payload.Add("scope", "admin:org repo") // todo: add more here

	req, _ := http.NewRequest("POST", url1, strings.NewReader(payload.Encode()))

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", pkg.JSONContentType)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var device Device1
	err := json.Unmarshal(body, &device)
	if err != nil {
		log.Println(err)
	}

	fmt.Println("---debug---")
	fmt.Println(device.UserCode)
	fmt.Println(device.VerificationUri)
	fmt.Println("---debug---")

	// try X times, if timeout, fail/ context?
	for {
		clientId := "cfe20fec21fd8126d9be"
		x := handler.service.PoolAccessToken(clientId, device.DeviceCode)
		fmt.Println(x)
		fmt.Println("sleeping...")
		time.Sleep(5 * time.Second)

	}
	return nil
}
