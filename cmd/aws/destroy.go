/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
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

func destroyAws(cmd *cobra.Command, args []string) error {
	helpers.DisplayLogHints()

	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")
	cloudRegionFlag := viper.GetString("flags.cloud-region")
	// _, err := helpers.EvalDestroy(awsinternal.CloudProvider, gitProvider)
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

	log.Info().Msg("destroying kubefirst platform in aws")

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClientNoSSL := http.Client{Transport: customTransport}

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	atlantisWebhookURL := fmt.Sprintf("https://atlantis.%s/events", domainName)

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
	config := providerConfigs.GetConfig(clusterName, domainName, gitProvider, cGitOwner, gitProtocol)

	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
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
			// tfEnvs = awsinternal.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITHUB_TOKEN"] = cGitToken
			tfEnvs["GITHUB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
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
			// tfEnvs = awsinternal.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GITLAB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(gitlabClient.ParentGroupID)
			tfEnvs["TF_VAR_gitlab_owner"] = viper.GetString("flags.gitlab-owner")
			err = terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
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

	// // Retrieve ingress-nginx load balancer for deletion
	// // todo: There could be more load balancers than this if users add more.
	// awsClient := &awsinternal.Conf
	//
	// // Remove security groups to prevent hanging resources
	// // err = awsClient.DeleteEKSSecurityGroups(clusterName)
	// // if err != nil {
	// // 	log.Warn().Msgf("security groups for cluster %s not found: %s", clusterName, err)
	// // }
	//
	// // Remove ELBs to prevent hanging resources
	// log.Info().Msgf("getting elastic load balancer details for cluster %s", clusterName)
	// params, err := awsClient.GetLoadBalancersForDeletion(clusterName)
	// if err != nil {
	// 	return err
	// }
	// if len(params) == 0 {
	// 	log.Warn().Msgf("elastic load balancer for cluster %s not found, continuing", clusterName)
	// } else {
	// 	for _, lb := range params {
	// 		// Delete security groups first
	// 		for _, sg := range lb.ElbSourceSecurityGroups {
	// 			err := awsClient.DeleteSecurityGroup(sg)
	// 			if err != nil {
	// 				log.Error().Msgf("error removing security group %s: %s", sg, err)
	// 			}
	// 		}
	// 		// Delete Elastic Load Balancer
	// 		err = awsClient.DeleteElasticLoadBalancer(lb)
	// 		if err != nil {
	// 			log.Warn().Msgf("could not delete load balancer %s: %s", lb.ElbName, err)
	// 		}
	// 	}
	// }

	// this should only run if a cluster was created
	if viper.GetBool("kubefirst-checks.aws-eks-cluster-created") {
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(cloudRegionFlag),
		}))
		log.Info().Msgf("region %s", *sess.Config.Region)

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

		// Remove applications with external dependencies
		removeArgoCDApps := []string{"ingress-nginx-components", "ingress-nginx"}
		err = argocd.ArgoCDApplicationCleanup(clientset, removeArgoCDApps)
		if err != nil {
			log.Error().Msgf("encountered error during argocd application cleanup: %s")
		}
		// Pause before cluster destroy to prevent a race condition
		log.Info().Msg("waiting for argocd application deletion to complete...")
		time.Sleep(time.Second * 20)

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

		secData, err := k8s.ReadSecretV2(clientset, "argocd", "argocd-initial-admin-secret")
		if err != nil {
			return err
		}
		argocdPassword := secData["password"]

		argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
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

		// Pause before cluster destroy to prevent a race condition
		log.Info().Msg("waiting for aws kubernetes cluster resource removal to finish...")
		time.Sleep(time.Second * 10)

		viper.Set("kubefirst-checks.aws-eks-cluster-created", false)
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-aws") || viper.GetBool("kubefirst-checks.terraform-apply-aws-failed") {
		log.Info().Msg("destroying aws resources with terraform")

		log.Info().Msg("destroying aws cloud resources")
		tfEntrypoint := config.GitopsDir + "/terraform/aws"
		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_aws_account_id"] = "awsAccountID"
		tfEnvs["TF_VAR_hosted_zone_name"] = domainNameFlag
		tfEnvs["AWS_SDK_LOAD_CONFIG"] = "1"
		tfEnvs["TF_VAR_aws_region"] = cloudRegionFlag
		tfEnvs["AWS_REGION"] = cloudRegionFlag
		err := terraform.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			viper.Set("kubefirst-checks.terraform-apply-aws-failed", true)
			viper.WriteConfig()
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
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", gitProvider)) && !viper.GetBool("kubefirst-checks.terraform-apply-aws") {
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
	fmt.Printf("Your kubefirst platform running in %s has been destroyed.", awsinternal.CloudProvider)

	return nil
}
