/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	b64 "encoding/base64"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/kubefirst/nebulous/pkg/flare"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		metricName := "kubefirst.mgmt_cluster_install.started"
		metricDomain := viper.GetString("aws.domainname")

		flare.SendTelemetry(metricDomain, metricName)

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)

		applyBase := viper.GetBool("create.terraformapplied.base")
		createSoftServeFlag := viper.GetBool("create.softserve.create")
		configureAndPushFlag := viper.GetBool("create.softserve.configure")

		if applyBase != true {

			terraformAction := "apply"

			os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
			os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
			os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.domainname"))

			err := os.Chdir(directory)
			if err != nil {
				fmt.Println("error changing dir")
			}

			viperDestoryFlag := viper.GetBool("terraform.destroy")
			cmdDestroyFlag, _ := cmd.Flags().GetBool("destroy")

			if viperDestoryFlag == true || cmdDestroyFlag == true {
				terraformAction = "destroy"
			}

			fmt.Println("terraform action: ", terraformAction, "destroyFlag: ", viperDestoryFlag)
			tfInitCmd := exec.Command(terraformPath, "init")
			tfInitCmd.Stdout = os.Stdout
			tfInitCmd.Stderr = os.Stderr
			err = tfInitCmd.Run()
			if err != nil {
				fmt.Println("failed to call tfInitCmd.Run(): ", err)
			}
			tfApplyCmd := exec.Command(terraformPath, fmt.Sprintf("%s", terraformAction), "-auto-approve")
			tfApplyCmd.Stdout = os.Stdout
			tfApplyCmd.Stderr = os.Stderr
			err = tfApplyCmd.Run()
			if err != nil {
				fmt.Println("failed to call tfApplyCmd.Run(): ", err)
				panic("tfApplyCmd.Run() failed")
			}
			keyIdBytes, err := exec.Command(terraformPath, "output", "vault_unseal_kms_key").Output()
			if err != nil {
				fmt.Println("failed to call tfOutputCmd.Run(): ", err)
			}
			keyId := strings.TrimSpace(string(keyIdBytes))

			fmt.Println("keyid is:", keyId)
			viper.Set("vault.kmskeyid", keyId)
			viper.Set("create.terraformapplied.base", true)
			viper.WriteConfig()

			detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))

		}
		if createSoftServeFlag != true {
			createSoftServe(kubeconfigPath)
			viper.Set("create.softserve.create", true)
			viper.WriteConfig()
			fmt.Println("waiting for soft-serve installation to complete...")
			time.Sleep(60 * time.Second)

		}

		if configureAndPushFlag != true {
			kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
			kPortForward.Stdout = os.Stdout
			kPortForward.Stderr = os.Stderr
			err := kPortForward.Start()
			defer kPortForward.Process.Signal(syscall.SIGTERM)
			if err != nil {
				fmt.Println("failed to call kPortForward.Run(): ", err)
			}
			time.Sleep(10 * time.Second)

			configureSoftServe()
			pushGitopsToSoftServe()
			viper.Set("create.softserve.configure", true)
			viper.WriteConfig()
		}

		time.Sleep(10 * time.Second)

		helmInstallArgocd(home, kubeconfigPath)
		awaitGitlab()

		fmt.Println("discovering gitlab toolbox pod")

		var outb, errb bytes.Buffer
		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "get", "pod", "-lapp=toolbox", "-o", "jsonpath='{.items[0].metadata.name}'")
		k.Stdout = &outb
		k.Stderr = &errb
		err := k.Run()
		if err != nil {
			fmt.Println("failed to call k.Run() to get gitlab pod: ", err)
		}
		gitlabPodName := outb.String()
		gitlabPodName = strings.Replace(gitlabPodName, "'", "", -1)
		fmt.Println("gitlab pod", gitlabPodName)

		gitlabToken := viper.GetString("gitlab.token")
		if gitlabToken == "" {

			fmt.Println("getting gitlab personal access token")

			id := uuid.New()
			gitlabToken = id.String()[:20]

			k = exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "exec", gitlabPodName, "--", "gitlab-rails", "runner", fmt.Sprintf("token = User.find_by_username('root').personal_access_tokens.create(scopes: [:write_registry, :write_repository, :api], name: 'Automation token'); token.set_token('%s'); token.save!", gitlabToken))
			k.Stdout = os.Stdout
			k.Stderr = os.Stderr
			err = k.Run()
			if err != nil {
				fmt.Println("failed to call k.Run() to set gitlab token: ", err)
			}

			viper.Set("gitlab.token", gitlabToken)
			viper.WriteConfig()

			fmt.Println("gitlabToken", gitlabToken)
		}

		gitlabRunnerToken := viper.GetString("gitlab.runnertoken")
		if gitlabRunnerToken == "" {

			fmt.Println("getting gitlab runner token")

			var tokenOut, tokenErr bytes.Buffer
			k = exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "get", "secret", "gitlab-gitlab-runner-secret", "-o", "jsonpath='{.data.runner-registration-token}'")
			k.Stdout = &tokenOut
			k.Stderr = &tokenErr
			err = k.Run()
			if err != nil {
				fmt.Println("failed to call k.Run() to get gitlabRunnerRegistrationToken: ", err)
			}
			encodedToken := tokenOut.String()
			fmt.Println(encodedToken)
			encodedToken = strings.Replace(encodedToken, "'", "", -1)
			fmt.Println(encodedToken)
			gitlabRunnerRegistrationTokenBytes, err := base64.StdEncoding.DecodeString(encodedToken)
			gitlabRunnerRegistrationToken := string(gitlabRunnerRegistrationTokenBytes)
			fmt.Println(gitlabRunnerRegistrationToken)
			if err != nil {
				panic(err)
			}
			viper.Set("gitlab.runnertoken", gitlabRunnerRegistrationToken)
			viper.WriteConfig()
			fmt.Println("gitlabRunnerRegistrationToken", gitlabRunnerRegistrationToken)
		}

		if !viper.GetBool("create.terraformapplied.gitlab") {
			// Prepare for terraform gitlab execution
			os.Setenv("GITLAB_TOKEN", viper.GetString("gitlab.token"))
			os.Setenv("GITLAB_BASE_URL", fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname")))

			directory = fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)
			err = os.Chdir(directory)
			if err != nil {
				fmt.Println("error changing dir")
			}

			tfInitCmd := exec.Command(terraformPath, "init")
			tfInitCmd.Stdout = os.Stdout
			tfInitCmd.Stderr = os.Stderr
			err = tfInitCmd.Run()
			if err != nil {
				fmt.Println("failed to call tfInitCmd.Run(): ", err)
			}

			tfApplyCmd := exec.Command(terraformPath, "apply", "-auto-approve")
			tfApplyCmd.Stdout = os.Stdout
			tfApplyCmd.Stderr = os.Stderr
			err = tfApplyCmd.Run()
			if err != nil {
				fmt.Println("failed to call tfApplyCmd.Run(): ", err)
			}

			viper.Set("create.terraformapplied.gitlab", true)
			viper.WriteConfig()
		}

		// upload ssh public key
		if !viper.GetBool("gitlab.keyuploaded") {
			fmt.Println("uploading ssh public key to gitlab")
			data := url.Values{
				"title": {"kubefirst"},
				"key":   {viper.GetString("botpublickey")},
			}

			gitlabUrlBase := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname"))

			resp, err := http.PostForm(gitlabUrlBase+"/api/v4/user/keys?private_token="+gitlabToken, data)
			if err != nil {
				log.Fatal(err)
			}
			var res map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&res)
			fmt.Println(res)
			fmt.Println("ssh public key uploaded to gitlab")
			viper.Set("gitlab.keyuploaded", true)
			viper.WriteConfig()
		} else {
			fmt.Println("ssh public key already uploaded to gitlab")
		}

		pushGitopsToGitLab()
		changeRegistryToGitLab()
		configureVault()
		addGitlabOidcApplications()
		hydrateGitlabMetaphorRepo()
		metricName = "kubefirst.mgmt_cluster_install.completed"

		flare.SendTelemetry(metricDomain, metricName)
	},
}

func hydrateGitlabMetaphorRepo() {

	metaphorTemplateDir := fmt.Sprintf("%s/.kubefirst/metaphor", home)

	url := "https://github.com/kubefirst/metaphor-template"

	metaphorTemplateRepo, err := git.PlainClone(metaphorTemplateDir, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		panic("error cloning metaphor-template repo")
	}

	detokenize(metaphorTemplateDir)

	// todo make global
	domainName := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname"))
	fmt.Println("git remote add origin", domainName)
	_, err = metaphorTemplateRepo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "gitlab",
		URLs: []string{fmt.Sprintf("%s/kubefirst/metaphor.git", domainName)},
	})

	w, _ := metaphorTemplateRepo.Worktree()

	fmt.Println("Committing new changes...")
	w.Add(".")
	w.Commit("setting new remote upstream to gitlab", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: installerEmail,
			When:  time.Now(),
		},
	})

	err = metaphorTemplateRepo.Push(&git.PushOptions{
		RemoteName: "gitlab",
		Auth: &gitHttp.BasicAuth{
			Username: "root",
			Password: viper.GetString("gitlab.token"),
		},
	})
	if err != nil {
		fmt.Println("error pushing to remote", err)
	}

}

func changeRegistryToGitLab() {
	if !viper.GetBool("gitlab.registry") {

		type ArgocdGitCreds struct {
			PersonalAccessToken string
			URL                 string
			FullURL             string
		}

		pat := b64.StdEncoding.EncodeToString([]byte(viper.GetString("gitlab.token")))
		url := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("https://gitlab.%s/kubefirst/", viper.GetString("aws.domainname"))))
		fullurl := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.domainname"))))

		creds := ArgocdGitCreds{PersonalAccessToken: pat, URL: url, FullURL: fullurl}

		var argocdRepositoryAccessTokenSecret *v1.Secret
		kubeconfig := home + "/.kubefirst/gitops/terraform/base/kubeconfig_kubefirst"
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		argocdSecretClient = clientset.CoreV1().Secrets("argocd")

		var secrets bytes.Buffer

		c, err := template.New("creds-gitlab").Parse(`
      apiVersion: v1
      data:
        password: {{ .PersonalAccessToken }}
        url: {{ .URL }}
        username: cm9vdA==
      kind: Secret
      metadata:
        annotations:
          managed-by: argocd.argoproj.io
        labels:
          argocd.argoproj.io/secret-type: repo-creds
        name: creds-gitlab
        namespace: argocd
      type: Opaque
    `)
		if err := c.Execute(&secrets, creds); err != nil {
			log.Panic(err)
		}
		fmt.Println(secrets.String())

		ba := []byte(secrets.String())
		err = yaml.Unmarshal(ba, &argocdRepositoryAccessTokenSecret)

		_, err = argocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			panic(err)
		}

		var repoSecrets bytes.Buffer

		c, err = template.New("repo-gitlab").Parse(`
      apiVersion: v1
      data:
        project: ZGVmYXVsdA==
        type: Z2l0
        url: {{ .FullURL }}
      kind: Secret
      metadata:
        annotations:
          managed-by: argocd.argoproj.io
        labels:
          argocd.argoproj.io/secret-type: repository
        name: repo-gitlab
        namespace: argocd
      type: Opaque
    `)
		if err := c.Execute(&repoSecrets, creds); err != nil {
			log.Panic(err)
		}
		fmt.Println(repoSecrets.String())

		ba = []byte(repoSecrets.String())
		err = yaml.Unmarshal(ba, &argocdRepositoryAccessTokenSecret)

		_, err = argocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			panic(err)
		}

		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/.kubefirst/gitops/components/gitlab/argocd-adopts-gitlab.yaml", home))
		k.Stdout = os.Stdout
		k.Stderr = os.Stderr
		err = k.Run()
		if err != nil {
			fmt.Println("failed to call k.Run() to apply argocd patch to adopt gitlab: ", err)
		}

		viper.Set("gitlab.registry", true)
		viper.WriteConfig()
	}
}

func addGitlabOidcApplications() {
	domain := viper.GetString("aws.domainname")
	git, err := gitlab.NewClient(
		viper.GetString("gitlab.token"),
		gitlab.WithBaseURL(fmt.Sprintf("https://gitlab.%s/api/v4", domain)),
	)
	if err != nil {
		log.Fatal(err)
	}

	apps := []string{"argo", "argocd", "vault"}
	cb := make(map[string]string)
	cb["argo"] = fmt.Sprintf("https://argo.%s/oauth2/callback", domain)
	cb["argocd"] = fmt.Sprintf("https://argocd.%s/auth/callback", domain)
	cb["vault"] = fmt.Sprintf("https://vault.%s:8250/oidc/callback http://localhost:8250/oidc/callback https://vault.%s/ui/vault/auth/oidc/oidc/callback http://localhost:8200/ui/vault/auth/oidc/oidc/callback", domain, domain)

	for _, app := range apps {
		fmt.Println("checking to see if", app, "oidc application needs to be created in gitlab")
		appId := viper.GetString(fmt.Sprintf("gitlab.oidc.%s.applicationid", app))
		if appId == "" {

			// Create an application
			opts := &gitlab.CreateApplicationOptions{
				Name:        gitlab.String(app),
				RedirectURI: gitlab.String(cb[app]),
				Scopes:      gitlab.String("read_user openid email"),
			}
			createdApp, _, err := git.Applications.CreateApplication(opts)
			if err != nil {
				log.Fatal(err)
			}

			// List all applications
			existingApps, _, err := git.Applications.ListApplications(&gitlab.ListApplicationsOptions{})
			if err != nil {
				log.Fatal(err)
			}

			created := false
			for _, existingApp := range existingApps {
				if existingApp.ApplicationName == app {
					created = true
				}
			}
			if created {
				fmt.Println("created gitlab oidc application with applicationid", createdApp.ApplicationID)
				viper.Set(fmt.Sprintf("gitlab.oidc.%s.applicationid", app), createdApp.ApplicationID)
				viper.Set(fmt.Sprintf("gitlab.oidc.%s.secret", app), createdApp.Secret)

				secretData := map[string]interface{}{
					"data": map[string]interface{}{
						"application_id": createdApp.ApplicationID,
						"secret":         createdApp.Secret,
					},
				}
				secretPath := fmt.Sprintf("secret/data/oidc/%s", app)
				addVaultSecret(secretPath, secretData)
				viper.WriteConfig()
			} else {
				log.Panic("could not create gitlab iodc application", app)
			}
		}
	}
}

func addVaultSecret(secretPath string, secretData map[string]interface{}) {
	fmt.Println("vault called")

	config := vault.DefaultConfig()

	config.Address = fmt.Sprintf("https://vault.%s", viper.GetString("aws.domainname"))

	client, err := vault.NewClient(config)
	if err != nil {
		fmt.Println("unable to initialize Vault client: ", err)
	}

	client.SetToken(viper.GetString("vault.token"))

	// Writing a secret
	_, err = client.Logical().Write(secretPath, secretData)
	if err != nil {
		fmt.Println("unable to write secret: ", err)
	} else {
		fmt.Println("secret written successfully.")
	}
}

func configureVault() {
	if !viper.GetBool("create.terraformapplied.vault") {

		// ```
		// NOTE: the terraform here produces unnecessary $var.varname vars in the atlantis secret for nonsensitive values
		// the following atlantis secrets shouldn't have vars in the gitops source code for the atlantis secret, they
		// should look like us-east-1, in flat string code as non-sensitive vals - refactor soon.
		// "TF_VAR_aws_region": "us-east-1",
		// "TF_VAR_aws_account_id": "${var.aws_account_id}",
		// "TF_VAR_email_address": "${var.email_address}",
		// "TF_VAR_hosted_zone_id": "${var.hosted_zone_id}",
		// "TF_VAR_hosted_zone_name": "${var.hosted_zone_name}",
		// "TF_VAR_vault_addr": "${var.vault_addr}",
		// ```
		// ... obviously keep the sensitive values bound to vars

		var outb, errb bytes.Buffer
		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "get", "secret", "vault-unseal-keys", "-o", "jsonpath='{.data.cluster-keys\\.json}'")
		k.Stdout = &outb
		k.Stderr = &errb
		err := k.Run()
		if err != nil {
			fmt.Println("failed to call k.Run() to get gitlab pod: ", err)
		}
		vaultKeysEncoded := outb.String()
		vaultKeysEncoded = strings.Replace(vaultKeysEncoded, "'", "", -1)
		fmt.Println("vault keys", vaultKeysEncoded)

		vaultKeysBytes, err := base64.StdEncoding.DecodeString(vaultKeysEncoded)
		fmt.Println(vaultKeysBytes)
		if err != nil {
			panic(err)
		}
		vaultKeys := string(vaultKeysBytes)
		fmt.Println(vaultKeys)

		var dat map[string]interface{}
		if err := json.Unmarshal([]byte(vaultKeys), &dat); err != nil {
			panic(err)
		}
		vaultToken := dat["root_token"].(string)
		fmt.Println(vaultToken)
		viper.Set("vault.token", vaultToken)
		viper.WriteConfig()

		// Prepare for terraform vault execution
		os.Setenv("VAULT_ADDR", fmt.Sprintf("https://vault.%s", viper.GetString("aws.domainname")))
		os.Setenv("VAULT_TOKEN", vaultToken)

		os.Setenv("TF_VAR_vault_addr", fmt.Sprintf("https://vault.%s", viper.GetString("aws.domainname")))
		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_email_address", viper.GetString("adminemail"))
		os.Setenv("TF_VAR_gitlab_runner_token", viper.GetString("gitlab.runnertoken"))
		os.Setenv("TF_VAR_gitlab_token", viper.GetString("gitlab.token"))
		os.Setenv("TF_VAR_hosted_zone_id", viper.GetString("aws.domainid"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.domainname"))
		os.Setenv("TF_VAR_vault_token", viper.GetString("aws.domainname"))
		os.Setenv("TF_VAR_vault_redirect_uris", "[\"will-be-patched-later\"]")

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/vault", home)
		err = os.Chdir(directory)
		if err != nil {
			fmt.Println("error changing dir")
		}

		tfInitCmd := exec.Command(terraformPath, "init")
		tfInitCmd.Stdout = os.Stdout
		tfInitCmd.Stderr = os.Stderr
		err = tfInitCmd.Run()
		if err != nil {
			fmt.Println("failed to call vault terraform init: ", err)
		}

		tfApplyCmd := exec.Command(terraformPath, "apply", "-target", "module.bootstrap", "-auto-approve")
		tfApplyCmd.Stdout = os.Stdout
		tfApplyCmd.Stderr = os.Stderr
		err = tfApplyCmd.Run()
		if err != nil {
			fmt.Println("failed to call vault terraform apply: ", err)
		}

		viper.Set("create.terraformapplied.vault", true)
		viper.WriteConfig()
	}
}

func awaitGitlab() {

	fmt.Println("awaitGitlab called")
	max := 200
	for i := 0; i < max; i++ {

		// todo should this be aws.hostedzonedname since we're sticking to an
		// todo aws: and gcp: figure their nomenclature is more familar
		hostedZoneName := viper.GetString("aws.domainname")

		resp, _ := http.Get(fmt.Sprintf("https://gitlab.%s", hostedZoneName))
		if resp != nil && resp.StatusCode == 200 {
			fmt.Println("gitlab host resolved, 30 second grace period required...")
			time.Sleep(time.Second * 30)
			i = max
		} else {
			fmt.Println("gitlab host not resolved, sleeping 10s")
			time.Sleep(time.Second * 10)
		}
	}
}

func init() {
	nebulousCmd.AddCommand(createCmd)

	// createCmd.Flags().String("tf-entrypoint", "", "the entrypoint to execute the terraform from")
	// createCmd.MarkFlagRequired("tf-entrypoint")
	// todo make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")

}
