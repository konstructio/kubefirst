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
	"os"
	"os/exec"
	"strings"
	"time"

	b64 "encoding/base64"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
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

		if !dryrunMode {
			flare.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}
		

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		applyBaseTerraform(cmd,directory)
		createSoftServe(kubeconfigPath)
		configureSoftserveAndPush()		
		helmInstallArgocd(home, kubeconfigPath)
		awaitGitlab()
		produceGitlabTokens()
		applyGitlabTerraform(directory)
		gitlabKeyUpload()
		pushGitopsToGitLab()
		changeRegistryToGitLab()
		configureVault()
		addGitlabOidcApplications()
		hydrateGitlabMetaphorRepo()
		metricName = "kubefirst.mgmt_cluster_install.completed"
		
		if !dryrunMode {
			flare.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}
	},
}

func hydrateGitlabMetaphorRepo() {
	//TODO: Should this be skipped if already executed?
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, hydrateGitlabMetaphorRepo skipped.")
		return
	}
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
	log.Println("git remote add origin", domainName)
	_, err = metaphorTemplateRepo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "gitlab",
		URLs: []string{fmt.Sprintf("%s/kubefirst/metaphor.git", domainName)},
	})

	w, _ := metaphorTemplateRepo.Worktree()

	log.Println("Committing new changes...")
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
		log.Println("error pushing to remote", err)
	}

}

func changeRegistryToGitLab() {
	if !viper.GetBool("gitlab.registry") {
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, changeRegistryToGitLab skipped.")
			return
		}

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
		log.Println(secrets.String())

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
		log.Println(repoSecrets.String())

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
			log.Println("failed to call k.Run() to apply argocd patch to adopt gitlab: ", err)
		}

		viper.Set("gitlab.registry", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: changeRegistryToGitLab")
	}
}

func addGitlabOidcApplications() {
	//TODO: Should this skipped if already executed. 
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, addGitlabOidcApplications skipped.")
		return
	}
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
		log.Println("checking to see if", app, "oidc application needs to be created in gitlab")
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
				log.Println("created gitlab oidc application with applicationid", createdApp.ApplicationID)
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
	log.Println("vault called")

	config := vault.DefaultConfig()

	config.Address = fmt.Sprintf("https://vault.%s", viper.GetString("aws.domainname"))

	client, err := vault.NewClient(config)
	if err != nil {
		log.Println("unable to initialize Vault client: ", err)
	}

	client.SetToken(viper.GetString("vault.token"))

	// Writing a secret
	_, err = client.Logical().Write(secretPath, secretData)
	if err != nil {
		log.Println("unable to write secret: ", err)
	} else {
		log.Println("secret written successfully.")
	}
}

func configureVault() {
	if !viper.GetBool("create.terraformapplied.vault") {
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, configureVault skipped.")
			return
		}
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

		//TODO replace this command: 
		var outb, errb bytes.Buffer
		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "get", "secret", "vault-unseal-keys", "-o", "jsonpath='{.data.cluster-keys\\.json}'")
		k.Stdout = &outb
		k.Stderr = &errb
		err := k.Run()
		if err != nil {
			log.Println("failed to call k.Run() to get gitlab pod: ", err)
		}
		vaultKeysEncoded := outb.String()
		vaultKeysEncoded = strings.Replace(vaultKeysEncoded, "'", "", -1)
		log.Println("vault keys", vaultKeysEncoded)

		vaultKeysBytes, err := base64.StdEncoding.DecodeString(vaultKeysEncoded)
		log.Println(vaultKeysBytes)
		if err != nil {
			panic(err)
		}
		vaultKeys := string(vaultKeysBytes)
		log.Println(vaultKeys)

		var dat map[string]interface{}
		if err := json.Unmarshal([]byte(vaultKeys), &dat); err != nil {
			panic(err)
		}
		vaultToken := dat["root_token"].(string)
		log.Println(vaultToken)
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
			log.Println("error changing dir")
		}

		tfInitCmd := exec.Command(terraformPath, "init")
		tfInitCmd.Stdout = os.Stdout
		tfInitCmd.Stderr = os.Stderr
		err = tfInitCmd.Run()
		if err != nil {
			log.Println("failed to call vault terraform init: ", err)
		}

		tfApplyCmd := exec.Command(terraformPath, "apply", "-target", "module.bootstrap", "-auto-approve")
		tfApplyCmd.Stdout = os.Stdout
		tfApplyCmd.Stderr = os.Stderr
		err = tfApplyCmd.Run()
		if err != nil {
			log.Println("failed to call vault terraform apply: ", err)
		}

		viper.Set("create.terraformapplied.vault", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: configureVault")
	}
}

func awaitGitlab() {
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, awaitGitlab skipped.")
		return
	}
	log.Println("awaitGitlab called")
	max := 200
	for i := 0; i < max; i++ {

		// todo should this be aws.hostedzonedname since we're sticking to an
		// todo aws: and gcp: figure their nomenclature is more familar
		hostedZoneName := viper.GetString("aws.domainname")

		resp, _ := http.Get(fmt.Sprintf("https://gitlab.%s", hostedZoneName))
		if resp != nil && resp.StatusCode == 200 {
			log.Println("gitlab host resolved, 30 second grace period required...")
			time.Sleep(time.Second * 30)
			i = max
		} else {
			log.Println("gitlab host not resolved, sleeping 10s")
			time.Sleep(time.Second * 10)
		}
	}
}

func init() {
	rootCmd.AddCommand(createCmd)

	// createCmd.Flags().String("tf-entrypoint", "", "the entrypoint to execute the terraform from")
	// createCmd.MarkFlagRequired("tf-entrypoint")
	// todo make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.PersistentFlags().BoolVarP(&dryrunMode, "dry-run", "s", false, "set to dry-run mode, no changes done on cloud provider selected")

}
