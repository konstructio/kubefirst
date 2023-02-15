package argocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocdModel"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	yaml2 "gopkg.in/yaml.v2"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

var ArgocdSecretClient coreV1Types.SecretInterface

// todo call this ArgocdConfig or something not so generic
// Config ArgoCD configuration
type Config struct {
	Configs struct {
		Repositories struct {
			SoftServeGitops struct {
				URL      string `yaml:"url,omitempty"`
				Insecure string `json:"insecure,omitempty"`
				Type     string `json:"type,omitempty"`
				Name     string `json:"name,omitempty"`
			} `yaml:"soft-serve-gitops,omitempty"`
			RepoGitops struct {
				URL  string `yaml:"url,omitempty"`
				Type string `yaml:"type,omitempty"`
				Name string `yaml:"name,omitempty"`
			} `yaml:"github-serve-gitops,omitempty"`
		} `yaml:"repositories,omitempty"`
		CredentialTemplates struct {
			SSHCreds struct {
				URL           string `yaml:"url,omitempty"`
				SSHPrivateKey string `yaml:"sshPrivateKey,omitempty"`
			} `yaml:"ssh-creds,omitempty"`
		} `yaml:"credentialTemplates,omitempty"`
	} `yaml:"configs,omitempty"`
	Server struct {
		ExtraArgs []string `yaml:"extraArgs,omitempty"`
		Ingress   struct {
			Enabled     string `yaml:"enabled,omitempty"`
			Annotations struct {
				IngressKubernetesIoRewriteTarget   string `yaml:"ingress.kubernetes.io/rewrite-target,omitempty"`
				IngressKubernetesIoBackendProtocol string `yaml:"ingress.kubernetes.io/backend-protocol,omitempty"`
			} `yaml:"annotations,omitempty"`
			Hosts []string    `yaml:"hosts,omitempty"`
			TLS   []TLSConfig `yaml:"tls,omitempty"`
		} `yaml:"ingress,omitempty"`
	} `yaml:"server,omitempty"`
}

type TLSConfig struct {
	Hosts      []string `yaml:"hosts,omitempty"`
	SecretName string   `yaml:"secretName,omitempty"`
}

// SyncRetry tries to Sync ArgoCD as many times as requested by the attempts' parameter. On successful request, returns
// true and no error, on error, returns false and the reason it fails.
// Possible values for the ArgoCD status are Unknown and Synced, Unknown means the application has some error, and Synced
// means the application was synced successfully.
func SyncRetry(httpClient pkg.HTTPDoer, attempts int, interval int, applicationName string, token string) (bool, error) {

	for i := 0; i < attempts; i++ {

		httpCode, syncStatus, err := Sync(httpClient, applicationName, token)
		if err != nil {
			log.Error().Err(err).Msg("")
			return false, fmt.Errorf("unable to request ArgoCD Sync, error is: %v", err)
		}

		// success! ArgoCD is synced!
		if syncStatus == "Synced" {
			log.Info().Msg("ArgoCD application is synced")
			return true, nil
		}

		// keep trying
		if httpCode == http.StatusBadRequest {
			log.Info().Msg("another operation is already in progress")
		}

		log.Info().Msgf(
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

// create an argocd application
func CreateApplication(httpClient pkg.HTTPDoer, cascade, applicationName, argoCDToken string) (httpCodeResponse int, syncStatus string, Error error) {

	params := url.Values{}
	params.Add("cascade", cascade)
	paramBody := strings.NewReader(params.Encode())

	url := fmt.Sprintf("%s/api/v1/applications/%s", viper.GetString("argocd.local.service"), applicationName)
	log.Info().Msg(url)
	req, err := http.NewRequest(http.MethodDelete, url, paramBody)
	if err != nil {
		log.Error().Err(err).Msg("")
		return 0, "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("error sending DELETE request to ArgoCD for application (%s)", applicationName)
		return res.StatusCode, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Warn().Err(err).Msgf("argocd http response code is: %d", res.StatusCode)
		return res.StatusCode, "", nil
	}

	body, err := io.ReadAll(res.Body)
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

// Sync request ArgoCD to manual sync an application.
func DeleteApplication(httpClient pkg.HTTPDoer, applicationName, argoCDToken, cascade string) (httpCodeResponse int, syncStatus string, Error error) {

	params := url.Values{}
	params.Add("cascade", cascade)
	paramBody := strings.NewReader(params.Encode())

	url := fmt.Sprintf("%s/api/v1/applications/%s", viper.GetString("argocd.local.service"), applicationName)
	log.Info().Msg(url)
	req, err := http.NewRequest(http.MethodDelete, url, paramBody)
	if err != nil {
		log.Error().Err(err).Msg("")
		return 0, "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("error sending DELETE request to ArgoCD for application (%s)", applicationName)
		return res.StatusCode, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Warn().Err(err).Msgf("argocd http response code is: %d", res.StatusCode)
		return res.StatusCode, "", nil
	}

	body, err := io.ReadAll(res.Body)
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

func RefreshApplication(httpClient pkg.HTTPDoer, applicationName string, argoCDToken string) {

	url := fmt.Sprintf("%s/api/v1/applications?refresh=true", viper.GetString("argocd.local.service"))
	log.Debug().Msg(url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("error sending GET request to ArgoCD for refreshing application (%s)", applicationName)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("ArgoCD Sync response http code is: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	log.Debug().Msg(string(body))
}

func ListApplications(httpClient pkg.HTTPDoer, applicationName string, argoCDToken string) {

	url := fmt.Sprintf("%s/api/v1/applications", viper.GetString("argocd.local.service"))
	log.Debug().Msg(url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Warn().Msgf("error sending GET request to ArgoCD for refreshing application (%s)\n", applicationName)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Warn().Msgf("ArgoCD Sync response http code is: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	log.Debug().Msg(string(body))
}

// Sync request ArgoCD to manual sync an application.
func Sync(httpClient pkg.HTTPDoer, applicationName string, argoCDToken string) (httpCodeResponse int, syncStatus string, Error error) {

	url := fmt.Sprintf("%s/api/v1/applications/%s/sync", viper.GetString("argocd.local.service"), applicationName)
	log.Info().Msg(url)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("")
		return 0, "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))
	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("error sending POST request to ArgoCD for syncing application (%s)", applicationName)
		return res.StatusCode, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Warn().Err(err).Msgf("ArgoCD Sync response http code is: %d", res.StatusCode)
		return res.StatusCode, "", nil
	}

	body, err := io.ReadAll(res.Body)
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
		return "", errors.New("unable to retrieve argocd token")
	}

	body, err := io.ReadAll(res.Body)
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
		return "", errors.New("unable to retrieve argocd token, make sure provided credentials are valid")
	}

	return token, nil
}

// GetArgocdAuthToken issue token and retry in case of failure.
// todo: call the retry from outside of the function, and use GetArgoCDToken function to get token. At the moment there
// are two functions issuing tokens.
func GetArgocdAuthToken(dryRun bool) string {

	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, GetArgocdAuthToken skipped.")
		return "nothing"
	}

	time.Sleep(15 * time.Second)

	url := fmt.Sprintf("%s/api/v1/session", viper.GetString("argocd.local.service"))

	payload := strings.NewReader(fmt.Sprintf("{\n\t\"username\":\"admin\",\"password\":\"%s\"\n}", viper.GetString("argocd.admin.password")))

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Fatal().Err(err).Msg("error getting auth token from argocd")
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// N.B.: when used in production, also check for redirect loops
			return nil
		},
	}

	x := 20
	for i := 0; i < x; i++ {
		log.Info().Msgf("requesting auth token from argocd: attempt %d of %d", i+1, x)
		time.Sleep(5 * time.Second)
		res, err := client.Do(req)

		if err != nil {
			log.Print("error requesting auth token from argocd", err)
			continue
		} else {
			defer res.Body.Close()
			log.Debug().Msgf("Request ArgoCD Token: Result HTTP Status %d", res.StatusCode)
			if res.StatusCode != http.StatusOK {
				log.Print("HTTP status NOK")
				continue
			}
			body, err := io.ReadAll(res.Body)
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
				log.Warn().Msgf("error unmarshalling  %s", err)
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
	log.Panic().Msg("Fail to get a token")
	// This code is unreacheble, as in absence of token we want to fail the install.
	// I kept is to avoid compiler to complain.
	return ""
}

func SyncArgocdApplication(dryRun bool, applicationName, argocdAuthToken string) {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, SyncArgocdApplication skipped.")
		return
	}

	// todo need to replace this with a curl wrapper and see if it WORKS

	url := fmt.Sprintf("https://localhost:8080/api/v1/applications/%s/sync", applicationName)
	var outb bytes.Buffer

	_, _, err := pkg.ExecShellReturnStrings("curl", "-k", "-L", "-X", "POST", url, "-H", fmt.Sprintf("Authorization: Bearer %s", argocdAuthToken))
	log.Info().Msgf("the value from the curl command to sync registry in argocd is: %s", outb.String())
	if err != nil {
		log.Panic().Err(err).Msg("error: curl appSync failed failed")
	}
}

// ApplyRegistry - Apply Registry application
func ApplyRegistry(dryRun bool) error {
	config := configs.ReadConfig()
	if viper.GetBool("argocd.registry.applied") {
		log.Info().Msg("skipped ApplyRegistry - ")
		return nil
	}
	if !dryRun {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/helpers/registry-base.yaml", config.K1FolderPath))
		if err != nil {
			log.Warn().Msgf("failed to execute kubectl apply of registry-base: %s", err)
			return err
		}
		time.Sleep(45 * time.Second)
		viper.Set("argocd.registry.applied", true)
		viper.WriteConfig()
	}
	return nil
}

// KubectlCreateApplication - create an argocd application via `kubectl`
// todo should this be k1.Kubectl ?
func KubectlCreateApplication(kubeconfigPath, kubectlClientPath, registryYamlPath string) error {

	// if viper.GetBool("argocd.registry.applied") || dryRun {
	// 	log.Info().Msg("skipped ApplyRegistryLocal - ")
	// 	return nil
	// }

	_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
	if err != nil {
		log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
		return err
	}

	// viper.Set("argocd.registry.applied", true)
	// viper.WriteConfig()

	return nil
}

// CreateInitialArgoCDRepository - Fill and create `argocd-init-values.yaml` for GitHub installs.
// The `argocd-init-values.yaml` is applied during helm install.
func CreateInitialArgoCDRepository(argoConfig Config, k1Dir string) error {

	argoCdRepoYaml, err := yaml2.Marshal(&argoConfig)
	if err != nil {
		return fmt.Errorf("error: marshaling yaml for argo config %s", err)
	}

	err = os.WriteFile(fmt.Sprintf("%s/argocd-init-values.yaml", k1Dir), argoCdRepoYaml, 0644)
	if err != nil {
		return fmt.Errorf("error: could not write argocd-init-values.yaml %s", err)
	}
	viper.Set("argocd.initial-repository.created", true)

	err = viper.WriteConfig()
	if err != nil {
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
		log.Error().Err(err).Msg("")
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
		log.Error().Err(err).Msg("")
		return false, fmt.Errorf("IsAppSynched - Error checking if arcoCD app is synched")
	}
	log.Info().Msgf("App status: %s", app.Status.Sync.Status)

	if app.Status.Sync.Status == "Synced" {
		return true, nil
	}
	return false, nil
}

// todo: document it, deprecate the other waitArgoCDToBeReady
func WaitArgoCDToBeReady(dryRun bool, kubeconfigPath, kubectlClientVersion string) {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, waitArgoCDToBeReady skipped.")
		return
	}
	x := 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(kubectlClientVersion, "--kubeconfig", kubeconfigPath, "get", "namespace/argocd")
		if err != nil {
			log.Info().Msg("Waiting argocd to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("argocd namespace found, continuing")
			time.Sleep(5 * time.Second)
			break
		}
	}
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(kubectlClientVersion, "--kubeconfig", kubeconfigPath, "-n", "argocd", "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
		if err != nil {
			log.Info().Msg("Waiting for argocd pods to create, checking in 10 seconds")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("argocd pods found, waiting for them to be running")
			viper.Set("argocd.ready", true)
			viper.WriteConfig()
			time.Sleep(15 * time.Second)
			break
		}
	}
}

// GetArgoCDInitialLocalConfig build a Config struct for local installation
func GetArgoCDInitialLocalConfig(gitOpsRepo string, botPrivateKey string) Config {

	argoCDConfig := Config{}

	// Repo config
	argoCDConfig.Configs.Repositories.RepoGitops.URL = gitOpsRepo
	argoCDConfig.Configs.Repositories.RepoGitops.Type = "git"
	argoCDConfig.Configs.Repositories.RepoGitops.Name = "github-gitops"

	// Credentials
	argoCDConfig.Configs.CredentialTemplates.SSHCreds.URL = gitOpsRepo
	argoCDConfig.Configs.CredentialTemplates.SSHCreds.SSHPrivateKey = botPrivateKey

	// Ingress
	argoCDConfig.Server.ExtraArgs = []string{"--insecure"}
	argoCDConfig.Server.Ingress.Enabled = "true"
	argoCDConfig.Server.Ingress.Annotations.IngressKubernetesIoRewriteTarget = "/"
	argoCDConfig.Server.Ingress.Annotations.IngressKubernetesIoBackendProtocol = "HTTPS"
	argoCDConfig.Server.Ingress.Hosts = []string{"argocd.localdev.me"}

	argoCDConfig.Server.Ingress.TLS = []TLSConfig{
		{
			Hosts:      []string{"argocd.localdev.me"},
			SecretName: "argocd-tls",
		},
	}

	return argoCDConfig
}

// GetArgoCDInitialCloudConfig build a Config struct for Cloud installation
func GetArgoCDInitialCloudConfig(gitOpsRepo string, botPrivateKey string) Config {

	argoCDConfig := Config{}
	argoCDConfig.Configs.Repositories.RepoGitops.URL = gitOpsRepo
	argoCDConfig.Configs.Repositories.RepoGitops.Type = "git"
	argoCDConfig.Configs.Repositories.RepoGitops.Name = "github-gitops"
	argoCDConfig.Configs.CredentialTemplates.SSHCreds.URL = gitOpsRepo
	argoCDConfig.Configs.CredentialTemplates.SSHCreds.SSHPrivateKey = botPrivateKey

	return argoCDConfig
}
