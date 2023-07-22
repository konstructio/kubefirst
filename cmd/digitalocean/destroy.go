/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/progressPrinter"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyDigitalocean(cmd *cobra.Command, args []string) error {
	helpers.DisplayLogHints()

	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")
	// _, err := helpers.EvalDestroy(digitalocean.CloudProvider, gitProvider)
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

	log.Info().Msg("destroying kubefirst platform in digitalocean")

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")

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

	// Instantiate digitalocean config
	config := providerConfigs.GetConfig(
		clusterName,
		domainName,
		gitProvider,
		cGitOwner,
		gitProtocol,
		os.Getenv("CF_API_TOKEN"),
	)
	config.DigitaloceanToken = os.Getenv("DO_TOKEN")
	switch gitProvider {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
	}
	if len(config.DigitaloceanToken) == 0 {
		return fmt.Errorf("\n\nYour DO_TOKEN environment variable isn't set")
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
			tfEnvs = digitalocean.GetDigitaloceanTerraformEnvs(config, tfEnvs)
			tfEnvs = digitalocean.GetGithubTerraformEnvs(config, tfEnvs)
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
			tfEnvs = digitalocean.GetDigitaloceanTerraformEnvs(config, tfEnvs)
			tfEnvs = digitalocean.GetGitlabTerraformEnvs(config, tfEnvs, gitlabClient.ParentGroupID)
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

	// // this should only run if a cluster was created
	// if viper.GetBool("kubefirst-checks.digitalocean-kubernetes-cluster-created") {
	// 	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	// 	// Remove applications withku external dependencies
	// 	removeArgoCDApps := []string{
	// 		"ingress-nginx-components",
	// 		"ingress-nginx",
	// 		"argo-components",
	// 		"argo",
	// 		"atlantis-components",
	// 		"atlantis",
	// 		"vault-components",
	// 		"vault",
	// 	}
	// 	err = argocd.ArgoCDApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
	// 	if err != nil {
	// 		log.Error().Msgf("encountered error during argocd application cleanup: %s", err)
	// 	}
	// 	// Pause before cluster destroy to prevent a race condition
	// 	log.Info().Msg("waiting for argocd application deletion to complete...")
	// 	time.Sleep(time.Second * 20)

	// 	viper.Set("kubefirst-checks.digitalocean-kubernetes-cluster-created", false)
	// }

	// // Fetch cluster resources prior to deletion
	// digitaloceanConf := digitalocean.DigitaloceanConfiguration{
	// 	Client:  digitalocean.NewDigitalocean(config.DigitaloceanToken),
	// 	Context: context.Background(),
	// }
	// resources, err := digitaloceanConf.GetKubernetesAssociatedResources(clusterName)
	// if err != nil {
	// 	return err
	// }

	// if viper.GetBool("kubefirst-checks.terraform-apply-digitalocean") || viper.GetBool("kubefirst-checks.terraform-apply-digitalocean-failed") {
	// 	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	// 	log.Info().Msg("destroying digitalocean resources with terraform")

	// 	log.Info().Msg("opening argocd port forward")
	// 	//* ArgoCD port-forward
	// 	argoCDStopChannel := make(chan struct{}, 1)
	// 	defer func() {
	// 		close(argoCDStopChannel)
	// 	}()
	// 	k8s.OpenPortForwardPodWrapper(
	// 		kcfg.Clientset,
	// 		kcfg.RestConfig,
	// 		"argocd-server",
	// 		"argocd",
	// 		8080,
	// 		8080,
	// 		argoCDStopChannel,
	// 	)

	// 	log.Info().Msg("getting new auth token for argocd")

	// 	secData, err := k8s.ReadSecretV2(kcfg.Clientset, "argocd", "argocd-initial-admin-secret")
	// 	if err != nil {
	// 		return err
	// 	}
	// 	argocdPassword := secData["password"]

	// 	argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

	// 	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	// 	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	// 	argocdHttpClient := http.Client{Transport: customTransport}
	// 	log.Info().Msg("deleting the registry application")
	// 	httpCode, _, err := argocd.DeleteApplication(&argocdHttpClient, config.RegistryAppName, argocdAuthToken, "true")
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.Info().Msgf("http status code %d", httpCode)

	// 	// Pause before cluster destroy to prevent a race condition
	// 	log.Info().Msg("waiting for digitalocean Kubernetes cluster resource removal to finish...")
	// 	time.Sleep(time.Second * 10)

	// 	log.Info().Msg("destroying digitalocean cloud resources")
	// 	tfEntrypoint := config.GitopsDir + "/terraform/digitalocean"
	// 	tfEnvs := map[string]string{}
	// 	tfEnvs = digitalocean.GetDigitaloceanTerraformEnvs(config, tfEnvs)

	// 	switch gitProvider {
	// 	case "github":
	// 		tfEnvs = digitalocean.GetGithubTerraformEnvs(config, tfEnvs)
	// 	case "gitlab":
	// 		gid, err := strconv.Atoi(viper.GetString("flags.gitlab-owner-group-id"))
	// 		if err != nil {
	// 			return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
	// 		}
	// 		tfEnvs = digitalocean.GetGitlabTerraformEnvs(config, tfEnvs, gid)
	// 	}
	// 	err = terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
	// 	if err != nil {
	// 		log.Printf("error executing terraform destroy %s", tfEntrypoint)
	// 		return err
	// 	}
	// 	viper.Set("kubefirst-checks.terraform-apply-digitalocean", false)
	// 	viper.WriteConfig()
	// 	log.Info().Msg("digitalocean resources terraform destroyed")
	// 	progressPrinter.IncrementTracker("platform-destroy", 1)
	// }

	// // Remove hanging volumes
	// err = digitaloceanConf.DeleteKubernetesClusterVolumes(resources)
	// if err != nil {
	// 	return err
	// }

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
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.terraform-apply-digitalocean") {
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
	fmt.Printf("Your kubefirst platform running in %s has been destroyed.", digitalocean.CloudProvider)

	return nil
}
