package k3d

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/github"
	gitlab "github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyK3d(cmd *cobra.Command, args []string) error {
	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	_, err := helpers.EvalDestroy(k3d.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(9000)
	if err != nil {
		log.Fatal().Msgf("%s - this port is required to tear down your kubefirst environment - please close any existing port forwards before continuing", err.Error())
		return err
	}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 1)
	progressPrinter.AddTracker("platform-destroy", "Destroying your kubefirst platform", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("destroying kubefirst platform running in k3d")

	clusterName := viper.GetString("flags.cluster-name")
	atlantisWebhookURL := fmt.Sprintf("%s/events", viper.GetString("ngrok.host"))
	dryRun := viper.GetBool("flags.dry-run")

	// Switch based on git provider, set params
	var cGitOwner, cGitToken string
	switch gitProvider {
	case "github":
		cGitOwner = viper.GetString("flags.github-owner")
		cGitToken = viper.GetString("github.session_token")
	case "gitlab":
		cGitOwner = viper.GetString("flags.gitlab-owner")
		cGitToken = os.Getenv("GITLAB_TOKEN")
	default:
		log.Panic().Msgf("invalid git provider option")
	}

	// Instantiate K3d config
	config := k3d.GetConfig(gitProvider, cGitOwner)

	log.Info().Msg("destroying kubefirst platform running in k3d")
	restConfig, err := k8s.GetClientConfig(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := k8s.GetClientSet(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	// todo improve these checks, make them standard for
	// both create and destroy
	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
	}

	if viper.GetBool("kubefirst-checks.post-detokenize") {
		// Remove remaining webhooks
		configmap, err := k8s.ReadConfigMapV2(config.Kubeconfig, "atlantis", "ngrok")
		if err != nil {
			return err
		}
		webhookURL := configmap["active-ngrok-tunnel-url"]

		switch config.GitProvider {
		case "github":
			githubSession := github.New(cGitToken)
			err = githubSession.DeleteRepositoryWebhook(cGitOwner, "gitops", webhookURL)
			if err != nil {
				log.Error().Msgf("error removing webhook: %s - you may need to manually remove it", err)
			}
		case "gitlab":
			gitlabClient, err := gitlab.NewGitLabClient(cGitToken, cGitOwner)
			if err != nil {
				return err
			}
			err = gitlabClient.DeleteProjectWebhook("gitops", webhookURL)
			if err != nil {
				log.Error().Msgf("error removing webhook: %s - you may need to manually remove it", err)
			}
		}

		// Temporary func to allow destroy
		err = k3d.ResolveMinioLocal(fmt.Sprintf("%s/terraform", config.GitopsDir))
		if err != nil {
			log.Fatal().Msgf("error preloading files for terraform destroy: %s", err)
		}
		minioStopChannel := make(chan struct{}, 1)
		defer func() {
			close(minioStopChannel)
		}()
		k8s.OpenPortForwardPodWrapper(
			clientset,
			restConfig,
			"minio",
			"minio",
			9000,
			9000,
			minioStopChannel,
		)
	}

	progressPrinter.IncrementTracker("preflight-checks", 1)

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
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
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
			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	case "gitlab":
		if viper.GetBool("kubefirst-checks.terraform-apply-gitlab") {
			log.Info().Msg("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cGitToken, cGitOwner)
			if err != nil {
				return err
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			var projectsForDeletion = []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gitlabClient.CheckProjectExists(project)
				if err != nil {
					log.Fatal().Msgf("could not check for existence of project %s: %s", project, err)
				}
				if projectExists {
					log.Info().Msgf("checking project %s for container registries...", project)
					crr, err := gitlabClient.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						log.Fatal().Msgf("could not retrieve container registry repositories: %s", err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							err := gitlabClient.DeleteContainerRegistryRepository(project, cr.ID)
							if err != nil {
								log.Fatal().Msgf("error deleting container registry repository: %s", err)
							}
						}
					} else {
						log.Info().Msgf("project %s does not have any container registries, skipping", project)
					}
				} else {
					log.Info().Msgf("project %s does not exist, skipping", project)
				}
			}

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}

			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GITLAB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(gitlabClient.ParentGroupID)

			err = terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				return err
			}
			viper.Set("kubefirst-checks.terraform-apply-gitlab", false)
			viper.WriteConfig()
			log.Info().Msg("gitlab resources terraform destroyed")
			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-k3d") || viper.GetBool("kubefirst-checks.terraform-apply-k3d-failed") {
		log.Info().Msg("destroying k3d resources with terraform")

		err := k3d.DeleteK3dCluster(clusterName, config.K1Dir, config.K3dClient)
		if err != nil {
			return err
		}

		viper.Set("kubefirst-checks.terraform-apply-k3d", false)
		viper.WriteConfig()
		log.Info().Msg("k3d resources terraform destroyed")
		progressPrinter.IncrementTracker("platform-destroy", 1)
	}

	// remove ssh key provided one was created
	if viper.GetString("kbot.gitlab-user-based-ssh-key-title") != "" {
		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, cGitOwner)
		if err != nil {
			return err
		}
		log.Info().Msg("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey(viper.GetString("kbot.gitlab-user-based-ssh-key-title"))
		if err != nil {
			log.Warn().Msg(err.Error())
		}
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
	time.Sleep(time.Millisecond * 200) // allows progress bars to finish
	fmt.Printf("Your kubefirst platform running in %s has been destroyed.", k3d.CloudProvider)

	return nil
}
