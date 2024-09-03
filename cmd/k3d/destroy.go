/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	constants "github.com/konstructio/kubefirst-api/pkg/constants"
	gitlab "github.com/konstructio/kubefirst-api/pkg/gitlab"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	"github.com/konstructio/kubefirst-api/pkg/progressPrinter"
	"github.com/konstructio/kubefirst-api/pkg/terraform"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyK3d(_ *cobra.Command, _ []string) error {
	utils.DisplayLogHints()

	gitProvider := viper.GetString("flags.git-provider")
	clusterName := viper.GetString("flags.cluster-name")
	gitProtocol := viper.GetString("flags.git-protocol")

	if clusterName == "" {
		fmt.Printf("Your kubefirst platform running has been already destroyed.")
		progress.Progress.Quit()
	}

	if err := k8s.CheckForExistingPortForwards(9000); err != nil {
		return fmt.Errorf("%w - this port is required to tear down your kubefirst environment - please close any existing port forwards before continuing", err)
	}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 1)
	progressPrinter.AddTracker("platform-destroy", "Destroying your kubefirst platform", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("destroying kubefirst platform running in k3d")

	atlantisWebhookURL := fmt.Sprintf("%s/events", viper.GetString("ngrok.host"))

	var cGitOwner, cGitToken string
	switch gitProvider {
	case "github":
		cGitOwner = viper.GetString("flags.github-owner")
		cGitToken = viper.GetString("github.session_token")
	case "gitlab":
		cGitOwner = viper.GetString("flags.gitlab-owner")
		cGitToken = os.Getenv("GITLAB_TOKEN")
	default:
		return fmt.Errorf("invalid git provider option: %q", gitProvider)
	}

	config := k3d.GetConfig(clusterName, gitProvider, cGitOwner, gitProtocol)
	switch gitProvider {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
	}

	if viper.GetBool("kubefirst-checks.post-detokenize") {
		if err := k3d.ResolveMinioLocal(fmt.Sprintf("%s/terraform", config.GitopsDir)); err != nil {
			return fmt.Errorf("unable to preload files for terraform destroy: %w", err)
		}
		minioStopChannel := make(chan struct{}, 1)
		defer func() {
			close(minioStopChannel)
		}()

		k8s.OpenPortForwardPodWrapper(kcfg.Clientset, kcfg.RestConfig, "minio", "minio", 9000, 9000, minioStopChannel)
	}

	progressPrinter.IncrementTracker("preflight-checks", 1)

	switch gitProvider {
	case "github":
		if viper.GetBool("kubefirst-checks.terraform-apply-github") {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{
				"GITHUB_TOKEN":                        cGitToken,
				"GITHUB_OWNER":                        cGitOwner,
				"TF_VAR_atlantis_repo_webhook_secret": viper.GetString("secrets.atlantis-webhook"),
				"TF_VAR_atlantis_repo_webhook_url":    atlantisWebhookURL,
				"TF_VAR_kbot_ssh_public_key":          viper.GetString("kbot.public-key"),
				"AWS_ACCESS_KEY_ID":                   constants.MinioDefaultUsername,
				"AWS_SECRET_ACCESS_KEY":               constants.MinioDefaultPassword,
				"TF_VAR_aws_access_key_id":            constants.MinioDefaultUsername,
				"TF_VAR_aws_secret_access_key":        constants.MinioDefaultPassword,
			}

			if err := terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs); err != nil {
				return fmt.Errorf("unable to execute terraform destroy: %w", err)
			}
			viper.Set("kubefirst-checks.terraform-apply-github", false)
			viper.WriteConfig()
			log.Info().Msg("github resources terraform destroyed")
			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	case "gitlab":
		if viper.GetBool("kubefirst-checks.terraform-apply-gitlab") {
			log.Info().Msg("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cGitToken, cGitOwner)
			if err != nil {
				return fmt.Errorf("unable to create gitlab client: %w", err)
			}

			projectsForDeletion := []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gitlabClient.CheckProjectExists(project)
				if err != nil {
					return fmt.Errorf("unable to check existence of project %q: %w", project, err)
				}
				if projectExists {
					log.Info().Msgf("checking project %q for container registries...", project)
					crr, err := gitlabClient.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						return fmt.Errorf("unable to retrieve container registry repositories: %w", err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							if err := gitlabClient.DeleteContainerRegistryRepository(project, cr.ID); err != nil {
								return fmt.Errorf("unable to delete container registry repository %q: %w", cr.Path, err)
							}
						}
					} else {
						log.Info().Msgf("project %q does not have any container registries, skipping", project)
					}
				} else {
					log.Info().Msgf("project %q does not exist, skipping", project)
				}
			}

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{
				"GITLAB_TOKEN":                        cGitToken,
				"GITLAB_OWNER":                        cGitOwner,
				"TF_VAR_atlantis_repo_webhook_secret": viper.GetString("secrets.atlantis-webhook"),
				"TF_VAR_atlantis_repo_webhook_url":    atlantisWebhookURL,
				"TF_VAR_owner_group_id":               strconv.Itoa(gitlabClient.ParentGroupID),
			}

			if err := terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs); err != nil {
				return fmt.Errorf("unable to execute terraform destroy: %w", err)
			}

			viper.Set("kubefirst-checks.terraform-apply-gitlab", false)
			viper.WriteConfig()
			log.Info().Msg("gitlab resources terraform destroyed")
			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	}

	if viper.GetBool("kubefirst-checks.create-k3d-cluster") || viper.GetBool("kubefirst-checks.create-k3d-cluster-failed") {
		log.Info().Msg("destroying k3d resources with terraform")

		if err := k3d.DeleteK3dCluster(clusterName, config.K1Dir, config.K3dClient); err != nil {
			return fmt.Errorf("unable to delete k3d cluster %q: %w", clusterName, err)
		}

		viper.Set("kubefirst-checks.create-k3d-cluster", false)
		viper.WriteConfig()
		log.Info().Msg("k3d resources terraform destroyed")
		progressPrinter.IncrementTracker("platform-destroy", 1)
	}

	if viper.GetString("kbot.gitlab-user-based-ssh-key-title") != "" {
		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, cGitOwner)
		if err != nil {
			return fmt.Errorf("unable to create gitlab client for deleting ssh key: %w", err)
		}
		log.Info().Msg("attempting to delete managed ssh key...")
		if err := gitlabClient.DeleteUserSSHKey(viper.GetString("kbot.gitlab-user-based-ssh-key-title")); err != nil {
			log.Warn().Msg(err.Error())
		}
	}

	// * remove local content and kubefirst config file for re-execution
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		log.Info().Msg("removing previous platform content")

		if err := utils.ResetK1Dir(config.K1Dir); err != nil {
			return fmt.Errorf("unable to remove previous platform content: %w", err)
		}
		log.Info().Msg("previous platform content removed")

		log.Info().Msg("resetting `$HOME/.kubefirst` config")
		viper.Set("argocd", "")
		viper.Set(gitProvider, "")
		viper.Set("components", "")
		viper.Set("kbot", "")
		viper.Set("kubefirst-checks", "")
		viper.Set("kubefirst", "")
		viper.Set("flags", "")
		viper.WriteConfig()
	}

	if _, err := os.Stat(config.K1Dir + "/kubeconfig"); !os.IsNotExist(err) {
		if err := os.Remove(config.K1Dir + "/kubeconfig"); err != nil {
			return fmt.Errorf("unable to delete %q: %w", config.K1Dir+"/kubeconfig", err)
		}
	}
	time.Sleep(200 * time.Millisecond)
	fmt.Printf("Your kubefirst platform running in %q has been destroyed.", k3d.CloudProvider)
	progress.Progress.Quit()

	return nil
}
