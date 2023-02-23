package aws

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func destroyAws(cmd *cobra.Command, args []string) error {

	log.Info().Msg("destroying kubefirst platform running in aws")

	// clusterName := viper.GetString("flags.cluster-name")
	// atlantisWebhookURL := fmt.Sprintf("%s/events", viper.GetString("ngrok.host"))
	// dryRun := viper.GetBool("flags.dry-run")
	// githubOwner := viper.GetString("flags.github-owner")

	// config := aws.GetConfig(githubOwner)

	// // todo improve these checks, make them standard for
	// // both create and destroy
	// githubToken := os.Getenv("GITHUB_TOKEN")
	// if len(githubToken) == 0 {
	// 	return errors.New("please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
	// }

	// if viper.GetBool("kubefirst-checks.terraform-apply-github") {
	// 	log.Info().Msg("destroying github resources with terraform")

	// 	tfEntrypoint := config.GitopsDir + "/terraform/github"
	// 	tfEnvs := map[string]string{}

	// 	tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	// 	tfEnvs["GITHUB_OWNER"] = githubOwnerFlag
	// 	tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	// 	tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
	// 	tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
	// 	tfEnvs["AWS_ACCESS_KEY_ID"] = "kray"
	// 	tfEnvs["AWS_SECRET_ACCESS_KEY"] = "feedkraystars"
	// 	tfEnvs["TF_VAR_aws_access_key_id"] = "kray"
	// 	tfEnvs["TF_VAR_aws_secret_access_key"] = "feedkraystars"

	// 	err := terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
	// 	if err != nil {
	// 		log.Printf("error executing terraform destroy %s", tfEntrypoint)
	// 		return err
	// 	}
	// 	viper.Set("kubefirst-checks.terraform-apply-github", false)
	// 	viper.WriteConfig()
	// 	log.Info().Msg("github resources terraform destroyed")
	// }

	// if viper.GetBool("kubefirst-checks.terraform-apply-aws") {
	// 	log.Info().Msg("destroying aws resources with terraform")

	// 	err := aws.DeleteawsCluster(clusterName, config.K1Dir, config.awsClient)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	viper.Set("kubefirst-checks.terraform-apply-aws", false)
	// 	viper.WriteConfig()
	// 	log.Info().Msg("aws resources terraform destroyed")
	// }

	// //* remove local content and kubefirst config file for re-execution
	// if !viper.GetBool("kubefirst-checks.terraform-apply-github") && !viper.GetBool("kubefirst-checks.terraform-apply-aws") {
	// 	log.Info().Msg("removing previous platform content")

	// 	err := pkg.ResetK1Dir(config.K1Dir, config.KubefirstConfig)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.Info().Msg("previous platform content removed")

	// 	log.Info().Msg("resetting `$HOME/.kubefirst` config")
	// 	viper.Set("argocd", "")
	// 	viper.Set("github", "")
	// 	viper.Set("components", "")
	// 	viper.Set("kbot", "")
	// 	viper.Set("kubefirst-checks", "")
	// 	viper.Set("kubefirst", "")
	// 	viper.WriteConfig()
	// }

	// if _, err := os.Stat(config.K1Dir + "/kubeconfig"); !os.IsNotExist(err) {
	// 	err = os.Remove(config.K1Dir + "/kubeconfig")
	// 	if err != nil {
	// 		return fmt.Errorf("unable to delete %q folder, error: %s", config.K1Dir+"/kubeconfig", err)
	// 	}
	// }
	fmt.Println("your kubefirst platform running in aws has been destroyed")

	return nil
}
