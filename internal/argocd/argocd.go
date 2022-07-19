package argocd

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

	"github.com/kubefirst/kubefirst/pkg"
	"strings"
	"time"
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

func GetArgocdAuthToken(dryRun bool) string {

	if dryRun {
		log.Printf("[#99] Dry-run mode, GetArgocdAuthToken skipped.")
		return "nothing"
	}

	time.Sleep(15 * time.Second)

	url := fmt.Sprintf("%s/api/v1/session", viper.GetString("argocd.local.service"))

	payload := strings.NewReader(fmt.Sprintf("{\n\t\"username\":\"admin\",\"password\":\"%s\"\n}", viper.GetString("argocd.admin.password")))

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Fatal("error getting auth token from argocd ", err)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// N.B.: when used in production, also check for redirect loops
			return nil
		},
	}

	x := 3
	for i := 0; i < x; i++ {
		log.Print("requesting auth token from argocd: attempt %s of %s", i, x)
		time.Sleep(1 * time.Second)
		res, err := client.Do(req)
		
		if err != nil {
			log.Print("error requesting auth token from argocd", err)
			continue
		} else {	
			defer res.Body.Close()		
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Print("error sending POST request to get argocd auth token:", err)
				continue
			}

			var dat map[string]interface{}

			if err := json.Unmarshal(body, &dat); err != nil {
				log.Print("error unmarshalling  %s", err)
				continue
			}
			token := dat["token"]
			viper.Set("argocd.admin.apitoken", token)
			viper.WriteConfig()

			// todo clean this up later
			return token.(string)
		}
	}
	log.Panic("Fail to get a token")
	// This code is unreacheble, as in absence of token we want to fail the install.
	// I kept is to avoid compiler to complain. 
	return ""
}

func SyncArgocdApplication(dryRun bool, applicationName, argocdAuthToken string) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, SyncArgocdApplication skipped.")
		return
	}

	// todo need to replace this with a curl wrapper and see if it WORKS

	url := fmt.Sprintf("https://localhost:8080/api/v1/applications/%s/sync", applicationName)
	var outb bytes.Buffer

	_, _, err := pkg.ExecShellReturnStrings("curl", "-k", "-L", "-X", "POST", url, "-H", fmt.Sprintf("Authorization: Bearer %s", argocdAuthToken))
	log.Println("the value from the curl command to sync registry in argocd is:", outb.String())
	if err != nil {
		log.Panicf("error: curl appSync failed failed %s", err)
	}
}

func DeleteArgocdApplicationNoCascade(dryRun bool, applicationName, argocdAuthToken string) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, SyncArgocdApplication skipped.")
		return
	}

	// todo need to replace this with a curl wrapper and see if it WORKS

	url := fmt.Sprintf("https://localhost:8080/api/v1/applications/%s?cascade=false", applicationName)
	var outb bytes.Buffer

	_, _, err := pkg.ExecShellReturnStrings("curl", "-k", "-L", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", argocdAuthToken))
	log.Println("the value from the curl command to delete registry in argocd is:", outb.String())
	if err != nil {
		log.Panicf("error: curl app delete failed %s", err)
	}
}
