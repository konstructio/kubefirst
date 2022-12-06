package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"syscall"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
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
		log.Panic().Msgf("error: failed to port-forward to vault namespce svc/vault %s", err)
	}

	// Prepare for terraform vault execution
	envs := map[string]string{}

	if viper.GetString("git.mode") == "gitlab" {
		envs["TF_VAR_gitlab_runner_token"] = viper.GetString("gitlab.runnertoken")
		envs["TF_VAR_gitlab_token"] = viper.GetString("gitlab.token")
	}

	envs["VAULT_ADDR"] = "http://localhost:8200" //Should this come from init?
	envs["VAULT_TOKEN"] = vaultToken
	envs["AWS_SDK_LOAD_CONFIG"] = "1"

	aws.ProfileInjection(&envs)

	envs["AWS_DEFAULT_REGION"] = viper.GetString("aws.region")

	envs["TF_VAR_vault_addr"] = fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename"))
	envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
	envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
	envs["TF_VAR_email_address"] = viper.GetString("adminemail")
	envs["TF_VAR_github_token"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	envs["TF_VAR_hosted_zone_id"] = viper.GetString("aws.hostedzoneid") //# TODO: are we using this?
	envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")
	envs["TF_VAR_vault_token"] = vaultToken
	envs["TF_VAR_vault_redirect_uris"] = viper.GetString("vault.oidc_redirect_uris")
	envs["TF_VAR_git_provider"] = viper.GetString("git.mode")
	//envs["TF_VAR_ssh_private_key"] = viper.GetString("botprivatekey")
	//Escaping newline to allow certs to be loaded properly by terraform
	envs["TF_VAR_ssh_private_key"] = viper.GetString("botprivatekey")

	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botpublickey")

	directory := fmt.Sprintf("%s/gitops/terraform/vault", config.K1FolderPath)
	err = os.Chdir(directory)
	if err != nil {
		log.Panic().Msgf("error: could not change directory to " + directory)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		log.Panic().Msgf("error: terraform init failed %s", err)
	}
	if !viper.GetBool("create.terraformapplied.vault") {
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
		if err != nil {
			log.Panic().Msgf("error: terraform apply failed %s", err)
		}
		log.Info().Msgf("deleting the files found at", fmt.Sprintf("%s/.terraform/", directory))
		log.Info().Msgf("deleting the files found at", fmt.Sprintf("%s/.terraform.lock.hcl", directory))
		os.RemoveAll(fmt.Sprintf("%s/.terraform/", directory))
		os.RemoveAll(fmt.Sprintf("%s/.terraform.lock.hcl", directory))
		viper.Set("create.terraformapplied.vault", true)
		viper.WriteConfig()
	}
}

func addVaultSecret(secretPath string, secretData map[string]interface{}) {
	config := vault.DefaultConfig()
	config.Address = viper.GetString("vault.local.service")

	client, err := vault.NewClient(config)
	if err != nil {
		log.Panic().Err(err).Msg("unable to initialize vault client")
	}

	client.SetToken(viper.GetString("vault.token"))

	_, err = client.Logical().Write(secretPath, secretData)
	if err != nil {
		log.Panic().Msgf("unable to write secret vault secret %s - error: %s", secretPath, err)
	} else {
		log.Info().Msgf("secret successfully written to path: %s", secretPath)
	}
}

func GetOidcClientCredentials(dryRun bool) {

	if dryRun {
		log.Printf("[#99] Dry-run mode, GetOidcClientCredentials skipped.")
		return
	}

	oidcApps := []string{"argo", "argocd"}

	if viper.GetString("gitprovider") == "gitlab" {
		oidcApps = append(oidcApps, "gitlab")
	}

	config := vault.DefaultConfig()
	config.Address = viper.GetString("vault.local.service")

	client, err := vault.NewClient(config)
	if err != nil {
		log.Panic().Err(err).Msg("unable to initialize vault client")
	}

	client.SetToken(viper.GetString("vault.token"))

	for _, app := range oidcApps {

		secret, err := client.KVv2("secret").Get(context.TODO(), fmt.Sprintf("oidc/%s", app))
		if err != nil {
			log.Panic().Err(err).Msgf("Unable to read the oidc secrets from vault")
		}

		clientId, ok := secret.Data["client_id"].(string)
		clientSecret, ok := secret.Data["client_secret"].(string)

		if !ok {
			log.Fatal().Msgf(
				"value type assertion failed: %T %#v", secret.Data["client_id"], secret.Data["client_secret"],
			)
		}
		viper.Set(fmt.Sprintf("vault.oidc.%s.client_id", app), clientId)
		viper.Set(fmt.Sprintf("vault.oidc.%s.client_secret", app), clientSecret)
	}
	viper.WriteConfig()

}

func WaitVaultToBeRunning(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
		return
	}
	token := viper.GetString("vault.token")
	if len(token) == 0 {
		config := configs.ReadConfig()
		x := 50
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/vault")
			if err != nil {
				log.Info().Msg("Waiting vault to be born")
				time.Sleep(10 * time.Second)
			} else {
				log.Info().Msg("vault namespace found, continuing")
				time.Sleep(25 * time.Second)
				break
			}
		}

		//! failing
		x = 50
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "get", "pods", "-l", "app.kubernetes.io/instance=vault")
			if err != nil {
				log.Info().Msg("Waiting vault pods to create")
				time.Sleep(10 * time.Second)
			} else {
				log.Info().Msg("vault pods found, continuing")
				time.Sleep(15 * time.Second)
				break
			}
		}
	} else {
		log.Info().Msg("vault token arleady exists, skipping vault health checks waitVaultToBeRunning")
	}
}
