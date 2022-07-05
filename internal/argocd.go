package internal

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
)

// kSyncArgocdApplication request ArgoCD to manual sync an application. Expected parameters are the ArgoCD application
// name and ArgoCD token with enough permission to perform the request against Argo API. When the http request returns
// status 200 it means a successful request/true, any other http status response return false.
func kSyncArgocdApplication(applicationName, argocdAuthToken string) (bool, error) {

	// todo: instantiate a new client on every http request is bad idea, we might need to set a new architecture to avoid
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	// todo: url values can be stored on .env files, and consumed when necessary to avoid hardcode urls
	url := fmt.Sprintf("https://localhost:8081/api/v1/applications/%s/sync", applicationName)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Println(err)
		return false, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argocdAuthToken))
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error sending POST request to ArgoCD for syncing application (%s)\n", applicationName)
		return false, err
	}

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

type ArgoCDConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// getArgoCDToken expects ArgoCD username and password, and returns a ArgoCD Bearer Token.
func getArgoCDToken(username string, password string) (string, error) {

	// todo: instantiate a new client on every http request is bad idea, we might need to set a new architecture to avoid
	// todo: fast solution here is to have a singleton to avoid code duplication
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	// todo: move it to config
	url := "https://localhost:8080/api/v1/session"

	// todo: this is documentation only, delete it when there is some function calling it
	//Username: "admin",
	//Password: viper.GetString("argocd.admin.password"),

	argoCDConfig := ArgoCDConfig{
		Username: username,
		Password: password,
	}

	payload, err := json.Marshal(argoCDConfig)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.New("unable to retrieve ArgoCD token")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var jsonReturn map[string]interface{}
	err = json.Unmarshal(body, &jsonReturn)
	if err != nil {
		return "", err
	}
	token := fmt.Sprintf("%v", jsonReturn["token"])
	if len(token) == 0 {
		return "", errors.New("unable to retrieve ArgoCD token, make sure ArgoCD credentials are correct")
	}

	// update config file
	viper.Set("argocd.admin.apitoken", token)
	err = viper.WriteConfig()
	if err != nil {
		log.Println(err)
		return "", err
	}

	return token, nil
}
