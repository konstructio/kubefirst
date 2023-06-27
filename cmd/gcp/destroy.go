/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gcp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/gcp"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/progressPrinter"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyGCP(cmd *cobra.Command, args []string) error {
	helpers.DisplayLogHints()

	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	// _, err := helpers.EvalDestroy(gcp.CloudProvider, gitProvider)
	// if err != nil {
	// 	return err
	// }

	// Check for existing port forwards before continuing
	err := k8s.CheckForExistingPortForwards(8080)
	if err != nil {
		return fmt.Errorf("%s - this port is required to tear down your kubefirst environment - please close any existing port forwards before continuing", err.Error())
	}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("destroying kubefirst platform in gcp")

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	gcpProject := viper.GetString("flags.gcp-project")

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

	// Instantiate GCP config
	config := gcp.GetConfig(clusterName, domainName, gitProvider, cGitOwner)
	// This is the environment variable required to create and is set to the path of the service account json file
	// This gets read for terraform applies and is applied as a variable containing the contents of the file
	// This is otherwise leveraged by the runtime to provide application default credentials to the GCP go SDK/API
	config.GCPAuth = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	switch gitProvider {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	// todo improve these checks, make them standard for
	// both create and destroy
	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
	}
	if len(config.GCPAuth) == 0 {
		return fmt.Errorf("\n\nYour GOOGLE_CLOUD_KEYFILE_JSON environment variable isn't set")
	}
	progressPrinter.IncrementTracker("preflight-checks", 1)

	progressPrinter.AddTracker("platform-destroy", "Destroying your kubefirst platform", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	switch gitProvider {
	case "github":
		if viper.GetBool("kubefirst-checks.terraform-apply-github") {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = gcp.GetGithubTerraformEnvs(config, tfEnvs)
			a, _ := os.ReadFile(config.GCPAuth)
			tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
			tfEnvs["TF_VAR_project"] = gcpProject
			err := terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
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
			tfEnvs = gcp.GetGitlabTerraformEnvs(config, tfEnvs, gitlabClient.ParentGroupID)
			a, _ := os.ReadFile(config.GCPAuth)
			// Not sure why this breaks, adding it here for now
			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
			tfEnvs["TF_VAR_project"] = gcpProject
			err = terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				return err
			}
			viper.Set("kubefirst-checks.terraform-apply-gitlab", false)
			viper.WriteConfig()
			log.Info().Msg("github resources terraform destroyed")

			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-gcp") || viper.GetBool("kubefirst-checks.terraform-apply-gcp-failed") {
		log.Info().Msg("destroying gcp cloud resources")

		tfEntrypoint := config.GitopsDir + "/terraform/gcp"
		tfEnvs := map[string]string{}

		switch gitProvider {
		case "github":
			tfEnvs = gcp.GetGithubTerraformEnvs(config, tfEnvs)
		case "gitlab":
			gid, err := strconv.Atoi(viper.GetString("flags.gitlab-owner-group-id"))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = gcp.GetGitlabTerraformEnvs(config, tfEnvs, gid)
		}
		a, _ := os.ReadFile(config.GCPAuth)
		tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
		tfEnvs["TF_VAR_project"] = gcpProject

		err = terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		viper.Set("kubefirst-checks.terraform-apply-gcp", false)
		viper.WriteConfig()
		log.Info().Msg("gcp resources terraform destroyed")
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
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.terraform-apply-gcp") {
		log.Info().Msg("removing previous platform content")

		err := pkg.ResetK1Dir(config.K1Dir)
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
	time.Sleep(time.Second * 2) // allows progress bars to finish
	fmt.Printf("Your kubefirst platform running in %s has been destroyed.", gcp.CloudProvider)

	return nil
}
