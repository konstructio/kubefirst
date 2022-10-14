package argocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/spf13/viper"

	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocdModel"
	"github.com/kubefirst/kubefirst/pkg"
	yaml2 "gopkg.in/yaml.v2"
)

var ArgocdSecretClient coreV1Types.SecretInterface

// ConfigRepo - Sample config struct
type ConfigRepo struct {
	Configs struct {
		Repositories struct {
			RepoGitops struct {
				URL  string `yaml:"url"`
				Type string `yaml:"type"`
				Name string `yaml:"name"`
			} `yaml:"github-serve-gitops"`
		} `yaml:"repositories"`
		CredentialTemplates struct {
			SSHCreds struct {
				URL           string `yaml:"url"`
				SSHPrivateKey string `yaml:"sshPrivateKey"`
			} `yaml:"ssh-creds"`
		} `yaml:"credentialTemplates"`
	} `yaml:"configs"`
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

	var syncResponse argocdModel.V1alpha1Application
	err = json.Unmarshal(body, &syncResponse)
	if err != nil {
		return res.StatusCode, "", err
	}

	return res.StatusCode, syncResponse.Status.Sync.Status, nil
}

// GetArgoCDToken expects ArgoCD username and password, and returns a ArgoCD Bearer Token. ArgoCD username and password
// are stored in the viper file.
func GetArgoCDToken(username string, password string) (string, error) {

	// todo: top caller should receive the token, and then update the viper file outside of this function. This will
	// 		 help this functions to be more generic and can be used for different purposes.
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	url := pkg.ArgoCDLocalBaseURL + "/session"

	argoCDConfig := argocdModel.SessionSessionCreateRequest{
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

	// todo: top caller should receive the token, and then update the viper file outside of this function. This will
	// 		 help this functions to be more generic and can be used for different purposes.
	// update config file
	viper.Set("argocd.admin.apitoken", token)
	err = viper.WriteConfig()
	if err != nil {
		log.Println(err)
		return "", err
	}

	return token, nil
}

// GetArgocdAuthToken issue token and retry in case of failure.
// todo: call the retry from outside of the function, and use GetArgoCDToken function to get token. At the moment there
// are two functions issuing tokens.
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

	x := 20
	for i := 0; i < x; i++ {
		log.Printf("requesting auth token from argocd: attempt %d of %d", i+1, x)
		time.Sleep(5 * time.Second)
		res, err := client.Do(req)

		if err != nil {
			log.Print("error requesting auth token from argocd", err)
			continue
		} else {
			defer res.Body.Close()
			log.Printf("Request ArgoCD Token: Result HTTP Status %d", res.StatusCode)
			if res.StatusCode != http.StatusOK {
				log.Print("HTTP status NOK")
				continue
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Print("error sending POST request to get argocd auth token:", err)
				continue
			}

			var dat map[string]interface{}
			if body == nil {
				log.Print("body object is nil")
				continue
			}
			if err := json.Unmarshal(body, &dat); err != nil {
				log.Printf("error unmarshalling  %s", err)
				continue
			}
			if dat == nil {
				log.Print("dat object is nil")
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

// ApplyRegistry - Apply Registry application
func ApplyRegistry(dryRun bool) error {
	config := configs.ReadConfig()
	if viper.GetBool("argocd.registry.applied") {
		log.Println("skipped ApplyRegistry - ")
		return nil
	}
	if !dryRun {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/helpers/registry-base.yaml", config.K1FolderPath))
		if err != nil {
			log.Printf("failed to execute kubectl apply of registry-base: %s", err)
			return err
		}
		time.Sleep(45 * time.Second)
		viper.Set("argocd.registry.applied", true)
		viper.WriteConfig()
	}
	return nil
}

// ApplyRegistry - Apply Registry application
func ApplyRegistryLocal(dryRun bool) error {
	config := configs.ReadConfig()
	if viper.GetBool("argocd.registry.applied") {
		log.Println("skipped ApplyRegistry - ")
		return nil
	}
	if !dryRun {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/registry.yaml", config.K1FolderPath))
		if err != nil {
			log.Printf("failed to execute kubectl apply of registry-base: %s", err)
			return err
		}
		time.Sleep(45 * time.Second)
		viper.Set("argocd.registry.applied", true)
		viper.WriteConfig()
	}
	return nil
}

// CreateInitalArgoRepository - Fill and create argocd-init-values.yaml for Github installs
func CreateInitalArgoRepository(githubURL string) error {
	config := configs.ReadConfig()

	privateKey := viper.GetString("botprivatekey")

	argoConfig := ConfigRepo{}
	argoConfig.Configs.Repositories.RepoGitops.URL = githubURL
	argoConfig.Configs.Repositories.RepoGitops.Type = "git"
	argoConfig.Configs.Repositories.RepoGitops.Name = "github-gitops"
	argoConfig.Configs.CredentialTemplates.SSHCreds.URL = githubURL
	argoConfig.Configs.CredentialTemplates.SSHCreds.SSHPrivateKey = privateKey

	argoYaml, err := yaml2.Marshal(&argoConfig)
	if err != nil {
		log.Printf("error: marsheling yaml for argo config %s", err)
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath), argoYaml, 0644)
	if err != nil {
		log.Printf("error: could not write argocd-init-values.yaml %s", err)
		return err
	}
	return nil
}

// GetArgoCDApplication by receiving the ArgoCD token, and the application name, this function returns the full
// application data Application struct. This can be used when a resource needs to be updated, we firstly collect all
// Application data, update what is necessary and then request the PUT function to update the resource.
func GetArgoCDApplication(token string, applicationName string) (argocdModel.V1alpha1Application, error) {

	// todo: instantiate a new client on every http request isn't a good idea, we might want to work with methods and
	//       provide resources via structs.
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	url := pkg.ArgoCDLocalBaseURL + "/applications/" + applicationName
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	res, err := httpClient.Do(req)
	if err != nil {
		return argocdModel.V1alpha1Application{}, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return argocdModel.V1alpha1Application{}, err
	}

	var response argocdModel.V1alpha1Application
	err = json.Unmarshal(body, &response)
	if err != nil {
		return argocdModel.V1alpha1Application{}, err
	}

	return response, nil
}

// IsAppSynched - Verify if ArgoCD Application is in synch state
func IsAppSynched(token string, applicationName string) (bool, error) {
	app, err := GetArgoCDApplication(token, applicationName)
	if err != nil {
		log.Println(err)
		return false, fmt.Errorf("IsAppSynched - Error checking if arcoCD app is synched")
	}
	log.Println("App status:", app.Status.Sync.Status)

	if app.Status.Sync.Status == "Synced" {
		return true, nil
	}
	return false, nil
}
