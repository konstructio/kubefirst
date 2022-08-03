package argocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/viper"

	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
)

type ArgoCDConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SyncResponse struct {
	Status struct {
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
	}
}

// SyncRetry tries to Sync ArgoCD as many times as requested by the attempts' parameter. On successful request, returns
// true and no error, on error, returns false and the reason it fails.
// Possible values for the ArgoCD status are Unknown and Synced, Unknown means the application has some error, and Synced
// means the application was synced successfully.
func SyncRetry(httpClient pkg.HTTPDoer, attempts int, interval int, applicationName string, token string) (bool, error) {

	for i := 0; i < attempts; i++ {

		httpCode, syncStatus, err := Sync(httpClient, applicationName, token)
		if err != nil {
			log.Println(err)
			return false, fmt.Errorf("unable to request ArgoCD Sync, error is: %v", err)
		}

		// success! ArgoCD is synced!
		if syncStatus == "Synced" {
			log.Println("ArgoCD application is synced")
			return true, nil
		}

		// keep trying
		if httpCode == http.StatusBadRequest {
			log.Println("another operation is already in progress")
		}

		log.Printf(
			"(%d/%d) sleeping %d seconds before trying to ArgoCD sync again, last Sync status is: %q",
			i+1,
			attempts,
			interval,
			syncStatus,
		)
		time.Sleep(time.Duration(interval) * time.Second)
	}
	return false, nil
}

// Sync request ArgoCD to manual sync an application.
func Sync(httpClient pkg.HTTPDoer, applicationName string, argoCDToken string) (httpCodeResponse int, syncStatus string, Error error) {

	url := fmt.Sprintf("%s/api/v1/applications/%s/sync", viper.GetString("argocd.local.service"), applicationName)
	log.Println(url)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error sending POST request to ArgoCD for syncing application (%s)\n", applicationName)
		log.Println(err)
		return res.StatusCode, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("ArgoCD Sync response http code is: %d", res.StatusCode)
		return res.StatusCode, "", nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, "", err
	}

	var syncResponse SyncResponse
	err = json.Unmarshal(body, &syncResponse)
	if err != nil {
		return res.StatusCode, "", err
	}

	return res.StatusCode, syncResponse.Status.Sync.Status, nil
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

// todo: replace this functions with getArgoCDToken
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
		log.Printf("requesting auth token from argocd: attempt %d of %d", i+1, x)
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

func ApplyRegistry(dryRun bool) error {
	config := configs.ReadConfig()
	if !dryRun {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/helpers/registry-base.yaml", config.K1FolderPath))
		if err != nil {
			log.Printf("failed to execute kubectl apply of registry-base: %s", err)
			return err
		}
		_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/helpers/registry-github.yaml", config.K1FolderPath))
		if err != nil {
			log.Printf("failed to execute kubectl apply of registry-github: %s", err)
			return err
		}

		time.Sleep(45 * time.Second)
	}
	return nil
}
