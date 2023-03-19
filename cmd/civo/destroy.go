package civo

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/civo/civogo"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/civo"
	gitlab "github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyCivo(cmd *cobra.Command, args []string) error {
	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	_, err := helpers.EvalDestroy(civo.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(8080)
	if err != nil {
		return fmt.Errorf("%s - this port is required to tear down your kubefirst environment - please close any existing port forwards before continuing", err.Error())
	}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("destroying kubefirst platform in civo")

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	dryRun := viper.GetBool("flags.dry-run")

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

	// Instantiate civo config
	config := civo.GetConfig(clusterName, domainName, gitProvider, cGitOwner)

	// todo improve these checks, make them standard for
	// both create and destroy
	civoToken := os.Getenv("CIVO_TOKEN")

	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
	}
	if len(civoToken) == 0 {
		return errors.New("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security and set the environment variable")
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
			tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
			tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
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

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			var projectsForDeletion = []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gl.CheckProjectExists(project)
				if err != nil {
					log.Fatal().Msgf("could not check for existence of project %s: %s", project, err)
				}
				if projectExists {
					log.Info().Msgf("checking project %s for container registries...", project)
					crr, err := gl.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						log.Fatal().Msgf("could not retrieve container registry repositories: %s", err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							err := gl.DeleteContainerRegistryRepository(project, cr.ID)
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
			tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
			tfEnvs = civo.GetGitlabTerraformEnvs(tfEnvs, gid)
			err = terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				return err
			}
			viper.Set("kubefirst-checks.terraform-apply-gitlab", false)
			viper.WriteConfig()
			log.Info().Msg("github resources terraform destroyed")

			// Since groups are only marked for deletion, attempt to remove them permanently
			// This only works on < premium tiers
			groupsToDelete := []string{"admins", "developers"}
			for _, group := range groupsToDelete {
				gid, err := gl.GetGroupID(allgroups, group)
				if err != nil {
					log.Error().Msgf("could not get group id for group %s, skipping auto-remove: %s", group, err)
				}
				_, err = gl.Client.Groups.DeleteGroup(gid)
				if err != nil {
					log.Warn().Msgf("attempt to remove group %s (marked for deletion) failed - you will need to delete it manually: %s", group, err)
				}
			}

			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	}

	restConfig, err := k8s.GetClientConfig(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := k8s.GetClientSet(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		log.Info().Msg("destroying civo resources with terraform")

		clusterName := viper.GetString("flags.cluster-name")
		region := viper.GetString("flags.cloud-region")

		client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		cluster, err := client.FindKubernetesCluster(clusterName)
		if err != nil {
			return err
		}
		log.Info().Msg("cluster name: " + cluster.ID)

		clusterVolumes, err := client.ListVolumesForCluster(cluster.ID)
		if err != nil {
			return err
		}

		log.Info().Msg("opening argocd port forward")
		//* ArgoCD port-forward
		argoCDStopChannel := make(chan struct{}, 1)
		defer func() {
			close(argoCDStopChannel)
		}()
		k8s.OpenPortForwardPodWrapper(
			clientset,
			restConfig,
			"argocd-server",
			"argocd",
			8080,
			8080,
			argoCDStopChannel,
		)

		log.Info().Msg("getting new auth token for argocd")
		argocdAuthToken, err := argocd.GetArgoCDToken(viper.GetString("components.argocd.username"), viper.GetString("components.argocd.password"))
		if err != nil {
			return err
		}

		log.Info().Msgf("port-forward to argocd is available at %s", civo.ArgocdPortForwardURL)

		customTransport := http.DefaultTransport.(*http.Transport).Clone()
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		argocdHttpClient := http.Client{Transport: customTransport}
		log.Info().Msg("deleting the registry application")
		httpCode, _, err := argocd.DeleteApplication(&argocdHttpClient, config.RegistryAppName, argocdAuthToken, "true")
		if err != nil {
			return err
		}
		log.Info().Msgf("http status code %d", httpCode)

		for _, vol := range clusterVolumes {
			log.Info().Msg("removing volume with name: " + vol.Name)
			_, err := client.DeleteVolume(vol.ID)
			if err != nil {
				return err
			}
			log.Info().Msg("volume " + vol.ID + " deleted")
		}

		// Pause before cluster destroy to prevent a race condition
		log.Info().Msg("waiting for Civo Kubernetes cluster resource removal to finish...")
		time.Sleep(time.Second * 10)

		log.Info().Msg("destroying civo cloud resources")
		tfEntrypoint := config.GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)

		switch gitProvider {
		case "github":
			tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
		case "gitlab":
			gid, err := strconv.Atoi(viper.GetString("flags.gitlab-owner-group-id"))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = civo.GetGitlabTerraformEnvs(tfEnvs, gid)
		}
		err = terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		viper.Set("kubefirst-checks.terraform-apply-civo", false)
		viper.WriteConfig()
		log.Info().Msg("civo resources terraform destroyed")
		progressPrinter.IncrementTracker("platform-destroy", 1)
	}

	// remove ssh key provided one was created
	if viper.GetString("kbot.gitlab-user-based-ssh-key-title") != "" {
		gl := gitlab.GitLabWrapper{
			Client: gitlab.NewGitLabClient(cGitToken),
		}
		log.Info().Msg("attempting to delete managed ssh key...")
		err := gl.DeleteUserSSHKey(viper.GetString("kbot.gitlab-user-based-ssh-key-title"))
		if err != nil {
			log.Warn().Msg(err.Error())
		}
	}

	//* remove local content and kubefirst config file for re-execution
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.terraform-apply-civo") {
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
	time.Sleep(time.Second * 2) // allows progress bars to finish
	fmt.Printf("Your kubefirst platform running in %s has been destroyed.", civo.CloudProvider)

	return nil
}
