package k3d

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	gitlab "github.com/kubefirst/kubefirst/internal/gitlabcloud"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyK3d(cmd *cobra.Command, args []string) error {

	log.Info().Msg("destroying kubefirst platform running in k3d")

	clusterName := viper.GetString("flags.cluster-name")
	atlantisWebhookURL := fmt.Sprintf("%s/events", viper.GetString("ngrok.host"))
	dryRun := viper.GetBool("flags.dry-run")
	gitProvider := viper.GetString("flags.git-provider")

	// Switch based on git provider, set params
	var cGitOwner, cGitToken string
	switch gitProvider {
	case "github":
		cGitOwner = viper.GetString("flags.github-owner")
		cGitToken = os.Getenv("GITHUB_TOKEN")
	case "gitlab":
		cGitOwner = viper.GetString("flags.gitlab-owner")
		cGitToken = os.Getenv("GITLAB_TOKEN")
	default:
		log.Panic().Msgf("invalid git provider option")
	}

	// Instantiate K3d config
	config := k3d.GetConfig(gitProvider, cGitOwner)

	// todo improve these checks, make them standard for
	// both create and destroy
	if len(cGitToken) == 0 {
		return errors.New(
			fmt.Sprintf(
				"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
				strings.ToUpper(gitProvider), gitProvider,
			),
		)
	}

	switch gitProvider {
	case "github":
		if viper.GetBool("kubefirst-checks.terraform-apply-github") {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}

			tfEnvs["GITHUB_TOKEN"] = cGitToken
			tfEnvs["GITHUB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["AWS_ACCESS_KEY_ID"] = "kray"
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = "feedkraystars"
			tfEnvs["TF_VAR_aws_access_key_id"] = "kray"
			tfEnvs["TF_VAR_aws_secret_access_key"] = "feedkraystars"

			err := terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				return err
			}
			viper.Set("kubefirst-checks.terraform-apply-github", false)
			viper.WriteConfig()
			log.Info().Msg("github resources terraform destroyed")
		}
	case "gitlab":
		if viper.GetBool("kubefirst-checks.terraform-apply-gitlab") {
			log.Info().Msg("destroying gitlab resources with terraform")

			gl := gitlab.GitLabWrapper{
				Client: gitlab.NewGitLabClient(cGitToken),
			}
			allgroups, err := gl.GetGroups()
			if err != nil {
				log.Fatal().Msgf("could not read gitlab groups: %s", err)
			}
			gid, err := gl.GetGroupID(allgroups, cGitOwner)
			if err != nil {
				log.Fatal().Msgf("could not get group id for primary group: %s", err)
			}

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}

			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GITLAB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)

			err = terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				return err
			}
			viper.Set("kubefirst-checks.terraform-apply-gitlab", false)
			viper.WriteConfig()
			log.Info().Msg("gitlab resources terraform destroyed")
		}
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-k3d") {
		log.Info().Msg("destroying k3d resources with terraform")

		err := k3d.DeleteK3dCluster(clusterName, config.K1Dir, config.K3dClient)
		if err != nil {
			return err
		}

		viper.Set("kubefirst-checks.terraform-apply-k3d", false)
		viper.WriteConfig()
		log.Info().Msg("k3d resources terraform destroyed")
	}

	//* remove local content and kubefirst config file for re-execution
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.terraform-apply-k3d") {
		log.Info().Msg("removing previous platform content")

		err := pkg.ResetK1Dir(config.K1Dir, config.KubefirstConfig)
		if err != nil {
			return err
		}
		log.Info().Msg("previous platform content removed")

		log.Info().Msg("resetting `$HOME/.kubefirst` config")
		viper.Set("argocd", "")
		viper.Set(gitProvider, "")
		viper.Set("components", "")
		viper.Set("kbot", "")
		viper.Set("kubefirst-checks", "")
		viper.Set("kubefirst", "")
		viper.WriteConfig()
	}

	if _, err := os.Stat(config.K1Dir + "/kubeconfig"); !os.IsNotExist(err) {
		err = os.Remove(config.K1Dir + "/kubeconfig")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", config.K1Dir+"/kubeconfig", err)
		}
	}
	fmt.Println("your kubefirst platform running in k3d has been destroyed")

	return nil
}
