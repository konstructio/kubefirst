package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	vault "github.com/hashicorp/vault/api"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetVaultRootToken get `vault-unseal-keys` token on Vault.
func GetVaultRootToken(vaultSecretClient coreV1Types.SecretInterface) (string, error) {
	name := "vault-unseal-keys"
	log.Printf("Reading secret %s\n", name)
	secret, err := vaultSecretClient.Get(context.TODO(), name, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}

	var vaultRootToken string
	var jsonData map[string]interface{}
	for _, value := range secret.Data {
		if err := json.Unmarshal(value, &jsonData); err != nil {
			return "", err
		}
		vaultRootToken = jsonData["root_token"].(string)
	}
	return vaultRootToken, nil
}

func ConfigureVault(dryRun bool) {
	config := configs.ReadConfig()
	if !viper.GetBool("create.terraformapplied.vault") {
		if dryRun {
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

		vaultToken := viper.GetString("vault.token")
		var kPortForwardOutb, kPortForwardErrb bytes.Buffer
		kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
		kPortForward.Stdout = &kPortForwardOutb
		kPortForward.Stderr = &kPortForwardErrb
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("Commad Execution STDOUT: %s", kPortForwardOutb.String())
			log.Printf("Commad Execution STDERR: %s", kPortForwardErrb.String())
			log.Panicf("error: failed to port-forward to vault namespce svc/vault %s", err)
		}

		// Prepare for terraform vault execution
		envs := map[string]string{}
		envs["VAULT_ADDR"] = "http://localhost:8200" //Should this come from init?
		envs["VAULT_TOKEN"] = vaultToken
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["AWS_DEFAULT_REGION"] = viper.GetString("aws.region")

		envs["TF_VAR_vault_addr"] = fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename"))
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_email_address"] = viper.GetString("adminemail")
		envs["TF_VAR_gitlab_runner_token"] = viper.GetString("gitlab.runnertoken")
		envs["TF_VAR_gitlab_token"] = viper.GetString("gitlab.token")
		envs["TF_VAR_hosted_zone_id"] = viper.GetString("aws.domainid")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")
		envs["TF_VAR_vault_token"] = vaultToken
		envs["TF_VAR_vault_redirect_uris"] = "[\"will-be-patched-later\"]"

		directory := fmt.Sprintf("%s/gitops/terraform/vault", config.K1FolderPath)
		err = os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("error: terraform init failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-target", "module.bootstrap", "-auto-approve")
		if err != nil {
			log.Panicf("error: terraform apply failed %s", err)
		}

		viper.Set("create.terraformapplied.vault", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: configureVault")
	}
}

func AddGitlabOidcApplications(dryRun bool) {

	//TODO: Should this skipped if already executed.
	if dryRun {
		log.Printf("[#99] Dry-run mode, addGitlabOidcApplications skipped.")
		return
	}
	domain := viper.GetString("aws.hostedzonename")
	git, err := gitlab.NewClient(
		viper.GetString("gitlab.token"),
		gitlab.WithBaseURL(fmt.Sprintf("%s/api/v4", viper.GetString("gitlab.local.service"))),
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
				log.Panicf("error: could not list applications from gitlab")
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
				log.Panicf("could not create gitlab oidc application %s", app)
			}
		}
	}
}

func addVaultSecret(secretPath string, secretData map[string]interface{}) {
	config := vault.DefaultConfig()
	config.Address = viper.GetString("vault.local.service")

	client, err := vault.NewClient(config)
	if err != nil {
		log.Panicf("unable to initialize vault client %s", err)
	}

	client.SetToken(viper.GetString("vault.token"))

	_, err = client.Logical().Write(secretPath, secretData)
	if err != nil {
		log.Panicf("unable to write secret vault secret %s - error: %s", secretPath, err)
	} else {
		log.Println("secret successfully written to path: ", secretPath)
	}
}
