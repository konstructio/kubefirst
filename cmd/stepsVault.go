package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	vault "github.com/hashicorp/vault/api"
	internalVault "github.com/kubefirst/nebulous/internal/vault"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Panicf("error: getting config %s", err)
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Panicf("error: getting config %s", err)
		}

		vaultSecretClient = clientset.CoreV1().Secrets("vault")
		vaultToken, err := internalVault.GetVaultRootToken(vaultSecretClient)
		if err != nil {
			log.Panicf("unable to get vault root token, error: %s", err)
		}

		viper.Set("vault.token", vaultToken)
		viper.WriteConfig()

		// Prepare for terraform vault execution
		os.Setenv("VAULT_ADDR", viper.GetString("vault.local.service"))
		os.Setenv("VAULT_TOKEN", viper.GetString("vault.token"))
		os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
		os.Setenv("AWS_PROFILE", "starter") // todo this is an issue
		os.Setenv("AWS_DEFAULT_REGION", viper.GetString("aws.region"))

		os.Setenv("TF_VAR_vault_addr", viper.GetString("vault.local.service"))
		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_email_address", viper.GetString("adminemail"))
		os.Setenv("TF_VAR_gitlab_runner_token", viper.GetString("gitlab.runnertoken"))
		os.Setenv("TF_VAR_gitlab_token", viper.GetString("gitlab.token"))
		os.Setenv("TF_VAR_hosted_zone_id", viper.GetString("aws.domainid"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))
		os.Setenv("TF_VAR_vault_token", viper.GetString("vault.token"))
		os.Setenv("TF_VAR_vault_redirect_uris", "[\"will-be-patched-later\"]")

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/vault", home)
		err = os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		tfInitCmd := exec.Command(terraformPath, "init")
		tfInitCmd.Stdout = os.Stdout
		tfInitCmd.Stderr = os.Stderr
		err = tfInitCmd.Run()
		if err != nil {
			log.Panicf("error: terraform init failed %s", err)
		}

		tfApplyCmd := exec.Command(terraformPath, "apply", "-target", "module.bootstrap", "-auto-approve")
		tfApplyCmd.Stdout = os.Stdout
		tfApplyCmd.Stderr = os.Stderr
		err = tfApplyCmd.Run()
		if err != nil {
			log.Panicf("error: terraform apply failed %s", err)
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
	git, err := gitlab.NewClient(
		viper.GetString("gitlab.token"),
		gitlab.WithBaseURL(fmt.Sprintf("%s/api/v4", viper.GetString("gitlab.local.service"))),
	)
	if err != nil {
		log.Fatal(err)
	}

	domain := viper.GetString("aws.hostedzonename")
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
