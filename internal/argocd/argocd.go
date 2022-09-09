package argocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
	"net/http"

	"github.com/spf13/viper"

	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	yaml2 "gopkg.in/yaml.v2"
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

// Application is required with full specification since ArgoCD needs a PUT to update the syncPolicy, and there is no
// PATCH available
type Application struct {
	Metadata struct {
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		Uid               string    `json:"uid"`
		ResourceVersion   string    `json:"resourceVersion"`
		Generation        int       `json:"generation"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		ManagedFields     []struct {
			Manager    string    `json:"manager"`
			Operation  string    `json:"operation"`
			ApiVersion string    `json:"apiVersion"`
			Time       time.Time `json:"time"`
			FieldsType string    `json:"fieldsType"`
			FieldsV1   struct {
				FSpec struct {
					Field1 struct {
					} `json:"."`
					FDestination struct {
						Field1 struct {
						} `json:"."`
						FNamespace struct {
						} `json:"f:namespace"`
						FServer struct {
						} `json:"f:server"`
					} `json:"f:destination"`
					FProject struct {
					} `json:"f:project"`
					FSource struct {
						Field1 struct {
						} `json:"."`
						FPath struct {
						} `json:"f:path"`
						FRepoURL struct {
						} `json:"f:repoURL"`
					} `json:"f:source"`
					FSyncPolicy struct {
					} `json:"f:syncPolicy"`
				} `json:"f:spec,omitempty"`
				FStatus struct {
					Field1 struct {
					} `json:".,omitempty"`
					FHealth struct {
						FStatus struct {
						} `json:"f:status,omitempty"`
					} `json:"f:health"`
					FSummary struct {
						FImages struct {
						} `json:"f:images,omitempty"`
					} `json:"f:summary"`
					FSync struct {
						Field1 struct {
						} `json:".,omitempty"`
						FComparedTo struct {
							Field1 struct {
							} `json:".,omitempty"`
							FDestination struct {
								FNamespace struct {
								} `json:"f:namespace,omitempty"`
								FServer struct {
								} `json:"f:server,omitempty"`
							} `json:"f:destination"`
							FSource struct {
								FPath struct {
								} `json:"f:path,omitempty"`
								FRepoURL struct {
								} `json:"f:repoURL,omitempty"`
							} `json:"f:source"`
						} `json:"f:comparedTo"`
						FRevision struct {
						} `json:"f:revision,omitempty"`
						FStatus struct {
						} `json:"f:status,omitempty"`
					} `json:"f:sync"`
					FHistory struct {
					} `json:"f:history,omitempty"`
					FOperationState struct {
						Field1 struct {
						} `json:"."`
						FFinishedAt struct {
						} `json:"f:finishedAt"`
						FMessage struct {
						} `json:"f:message"`
						FOperation struct {
							Field1 struct {
							} `json:"."`
							FInitiatedBy struct {
								Field1 struct {
								} `json:"."`
								FUsername struct {
								} `json:"f:username"`
							} `json:"f:initiatedBy"`
							FRetry struct {
							} `json:"f:retry"`
							FSync struct {
								Field1 struct {
								} `json:"."`
								FRevision struct {
								} `json:"f:revision"`
								FSyncStrategy struct {
									Field1 struct {
									} `json:"."`
									FHook struct {
									} `json:"f:hook"`
								} `json:"f:syncStrategy"`
							} `json:"f:sync"`
						} `json:"f:operation"`
						FPhase struct {
						} `json:"f:phase"`
						FStartedAt struct {
						} `json:"f:startedAt"`
						FSyncResult struct {
							Field1 struct {
							} `json:"."`
							FResources struct {
							} `json:"f:resources"`
							FRevision struct {
							} `json:"f:revision"`
							FSource struct {
								Field1 struct {
								} `json:"."`
								FPath struct {
								} `json:"f:path"`
								FRepoURL struct {
								} `json:"f:repoURL"`
							} `json:"f:source"`
						} `json:"f:syncResult"`
					} `json:"f:operationState,omitempty"`
					FReconciledAt struct {
					} `json:"f:reconciledAt,omitempty"`
					FResources struct {
					} `json:"f:resources,omitempty"`
					FSourceType struct {
					} `json:"f:sourceType,omitempty"`
				} `json:"f:status"`
			} `json:"fieldsV1"`
		} `json:"managedFields"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			RepoURL string `json:"repoURL"`
			Path    string `json:"path"`
		} `json:"source"`
		Destination struct {
			Server    string `json:"server"`
			Namespace string `json:"namespace"`
		} `json:"destination"`
		Project    string `json:"project"`
		SyncPolicy struct {
		} `json:"syncPolicy"`
	} `json:"spec"`
	Status struct {
		Resources []struct {
			Version   string `json:"version"`
			Kind      string `json:"kind"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			Health    struct {
				Status  string `json:"status"`
				Message string `json:"message,omitempty"`
			} `json:"health"`
			Group string `json:"group,omitempty"`
		} `json:"resources"`
		Sync struct {
			Status     string `json:"status"`
			ComparedTo struct {
				Source struct {
					RepoURL string `json:"repoURL"`
					Path    string `json:"path"`
				} `json:"source"`
				Destination struct {
					Server    string `json:"server"`
					Namespace string `json:"namespace"`
				} `json:"destination"`
			} `json:"comparedTo"`
			Revision string `json:"revision"`
		} `json:"sync"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
		History []struct {
			Revision   string    `json:"revision"`
			DeployedAt time.Time `json:"deployedAt"`
			Id         int       `json:"id"`
			Source     struct {
				RepoURL string `json:"repoURL"`
				Path    string `json:"path"`
			} `json:"source"`
			DeployStartedAt time.Time `json:"deployStartedAt"`
		} `json:"history"`
		ReconciledAt   time.Time `json:"reconciledAt"`
		OperationState struct {
			Operation struct {
				Sync struct {
					Revision     string `json:"revision"`
					SyncStrategy struct {
						Hook struct {
						} `json:"hook"`
					} `json:"syncStrategy"`
				} `json:"sync"`
				InitiatedBy struct {
					Username string `json:"username"`
				} `json:"initiatedBy"`
				Retry struct {
				} `json:"retry"`
			} `json:"operation"`
			Phase      string `json:"phase"`
			Message    string `json:"message"`
			SyncResult struct {
				Resources []struct {
					Group     string `json:"group"`
					Version   string `json:"version"`
					Kind      string `json:"kind"`
					Namespace string `json:"namespace"`
					Name      string `json:"name"`
					Status    string `json:"status"`
					Message   string `json:"message"`
					HookPhase string `json:"hookPhase"`
					SyncPhase string `json:"syncPhase"`
				} `json:"resources"`
				Revision string `json:"revision"`
				Source   struct {
					RepoURL string `json:"repoURL"`
					Path    string `json:"path"`
				} `json:"source"`
			} `json:"syncResult"`
			StartedAt  time.Time `json:"startedAt"`
			FinishedAt time.Time `json:"finishedAt"`
		} `json:"operationState"`
		SourceType string `json:"sourceType"`
		Summary    struct {
			Images []string `json:"images"`
		} `json:"summary"`
	} `json:"status"`
}

var ArgocdSecretClient coreV1Types.SecretInterface

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

// GetArgoCDToken expects ArgoCD username and password, and returns a ArgoCD Bearer Token. ArgoCD username and password
// are stored in the viper file.
func GetArgoCDToken(username string, password string) (string, error) {

	// todo: top caller should receive the token, and then update the viper file outside of this function. This will
	// 		 help this functions to be more generic and can be used for different purposes.
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	url := pkg.ArgoCDLocalBaseURL + "/session"

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
//
//	are two functions issuing tokens.
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
		_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/helpers/registry-github.yaml", config.K1FolderPath))
		if err != nil {
			log.Printf("failed to execute kubectl apply of registry-github: %s", err)
			return err
		}

		time.Sleep(45 * time.Second)
		viper.Set("argocd.registry.applied", true)
		viper.WriteConfig()
	}
	return nil
}

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

// todo: make it generic function at pkg/ folder, it isn't a ArgoCD domain function
func AddArgoCDApp(gitopsDir string) error {
	sourceFile := gitopsDir + "/components/helpers/argocd.yaml"
	destinationFile := gitopsDir + "/registry/base/argocd.yaml"
	log.Println("Source file:", sourceFile)
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Println(err)
		return err
	}

	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		log.Println("Error creating", destinationFile)
		log.Println(err)
		return err
	}
	return nil
}

// GetArgoCDApplication by receiving the ArgoCD token, and the application name, this function returns the full
// application data Application struct. This can be used when a resource needs to be updated, we firstly collect all
// Application data, update what is necessary and then request the PUT function to update the resource.
func GetArgoCDApplication(token string, applicationName string) (Application, error) {

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
		return Application{}, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Application{}, err
	}

	var response Application
	err = json.Unmarshal(body, &response)
	if err != nil {
		return Application{}, err
	}

	return response, nil
}

// PutArgoCDApplication expects a ArgoCD token and filled Application struct, ArgoCD will receive the request, and
// update the deltas. Since this functions is calling via PUT http verb, Application needs to be filled properly to
// be able to reflect the changes. note: PUT is different from PATCH verb.
func PutArgoCDApplication(token string, argoCDApplication Application) error {

	// todo: instantiate a new client on every http request isn't a good idea, we might want to work with methods and
	//       provide resources via structs.
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	url := pkg.ArgoCDLocalBaseURL + "/applications/shockshop?validate=false"

	payload, err := json.Marshal(argoCDApplication)
	if err != nil {
		log.Println(err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		log.Println(err)
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := httpClient.Do(req)
	if err != nil {
		log.Println(err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to update ArgoCD application, http response code is %d", res.StatusCode)
	}

	return nil
}
