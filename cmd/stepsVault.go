package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/spf13/viper"
	"os/exec"
	"encoding/json"
	"bytes"
	"encoding/base64"
	gitlab "github.com/xanzy/go-gitlab"
	vault "github.com/hashicorp/vault/api"
)


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