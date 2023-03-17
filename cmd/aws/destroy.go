package aws

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/kubefirst/kubefirst/internal/argocd"
	awsinternal "github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/civo"
	gitlab "github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyAws(cmd *cobra.Command, args []string) error {

	log.Info().Msg("destroying kubefirst platform running in aws")

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClientNoSSL := http.Client{Transport: customTransport}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("destroying kubefirst platform in aws")

	clusterName := viper.GetString("flags.cluster-name")
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

	// Instantiate aws config
	config := awsinternal.GetConfig(cGitOwner)

	progressPrinter.IncrementTracker("preflight-checks", 1)

	progressPrinter.AddTracker("platform-destroy", "Destroying your kubefirst platform", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	switch gitProvider {
	case "github":
		if viper.GetBool("kubefirst-checks.terraform-apply-github") {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
			tfEnvs["GITHUB_OWNER"] = githubOwnerFlag
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = fmt.Sprintf("https://atlantis.%s/events", domainNameFlag)
			tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
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
			progressPrinter.IncrementTracker("platform-destroy", 1)
		}
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cloudRegionFlag),
	}))

	eksSvc := eks.New(sess)

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}
	eksClusterInfo, err := eksSvc.DescribeCluster(clusterInput)
	if err != nil {
		log.Fatal().Msgf("Error calling DescribeCluster: %v", err)
	}

	clientset, err := awsinternal.NewClientset(eksClusterInfo.Cluster)
	if err != nil {
		log.Fatal().Msgf("Error creating clientset: %v", err)
	}

	restConfig, err := awsinternal.NewRestConfig(eksClusterInfo.Cluster)
	if err != nil {
		return err
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-aws") {
		log.Info().Msg("destroying aws resources with terraform")

		if viper.GetBool("kubefirst-checks.argocd-helm-install") {
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

			log.Info().Msgf("port-forward to argocd is available at %s", pkg.ArgocdPortForwardURL)

			log.Info().Msg("deleting the registry application")
			httpCode, _, err := argocd.DeleteApplication(&httpClientNoSSL, pkg.RegistryAppName, argocdAuthToken, "true")
			if err != nil {
				return err
			}
			log.Info().Msgf("http status code %d", httpCode)

		}
		// Pause before cluster destroy to prevent a race condition
		log.Info().Msg("waiting for aws kubernetes cluster resource removal to finish...")
		time.Sleep(time.Second * 10)

		log.Info().Msg("destroying aws cloud resources")
		tfEntrypoint := config.GitopsDir + "/terraform/aws"
		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_aws_account_id"] = "awsAccountID"
		tfEnvs["TF_VAR_hosted_zone_name"] = domainNameFlag
		tfEnvs["AWS_SDK_LOAD_CONFIG"] = "1"
		tfEnvs["TF_VAR_aws_region"] = os.Getenv("AWS_REGION")
		tfEnvs["AWS_REGION"] = os.Getenv("AWS_REGION")

		switch gitProvider {
		case "github":
			tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
			tfEnvs["GITHUB_OWNER"] = githubOwnerFlag
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = "atlantisWebhookURL"
			tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
		case "gitlab":
			gid, err := strconv.Atoi(viper.GetString("flags.gitlab-owner-group-id"))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = civo.GetGitlabTerraformEnvs(tfEnvs, gid)
		}
		err := terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		viper.Set("kubefirst-checks.terraform-apply-aws", false)
		viper.WriteConfig()
		log.Info().Msg("aws resources terraform destroyed")
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
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.terraform-apply-aws") {
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
	fmt.Println("your kubefirst platform running in k3d has been destroyed")

	return nil
}
