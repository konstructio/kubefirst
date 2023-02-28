package aws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/services"
	internalssh "github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cancelContext context.CancelFunc
)

func createAws(cmd *cobra.Command, args []string) error {

	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		return err
	}

	clusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		return err
	}

	clusterTypeFlag, err := cmd.Flags().GetString("cluster-type")
	if err != nil {
		return err
	}

	domainNameFlag, err := cmd.Flags().GetString("domain-name")
	if err != nil {
		return err
	}

	dryRunFlag, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	githubOwnerFlag, err := cmd.Flags().GetString("github-owner")
	if err != nil {
		return err
	}

	gitopsTemplateURLFlag, err := cmd.Flags().GetString("gitops-template-url")
	if err != nil {
		return err
	}

	gitopsTemplateBranchFlag, err := cmd.Flags().GetString("gitops-template-branch")
	if err != nil {
		return err
	}

	kbotPasswordFlag, err := cmd.Flags().GetString("kbot-password")
	if err != nil {
		return err
	}

	metaphorTemplateURLFlag, err := cmd.Flags().GetString("metaphor-template-url")
	if err != nil {
		return err
	}

	metaphorTemplateBranchFlag, err := cmd.Flags().GetString("metaphor-template-branch")
	if err != nil {
		return err
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return err
	}

	httpClient := http.DefaultClient
	gitHubService := services.NewGitHubService(httpClient)
	gitHubHandler := handlers.NewGitHubHandler(gitHubService)

	// get github data to set user based on the provided token
	log.Info().Msg("verifying github authentication")
	// todo is this the correct lookup in an org setting
	// todo is this the correct lookup in an org setting
	_, err = gitHubHandler.GetGitHubUser(os.Getenv("GITHUB_TOKEN"))
	if err != nil {
		return err
	}

	// required for destroy command
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.dry-run", dryRunFlag)
	viper.Set("flags.github-owner", githubOwnerFlag)
	viper.WriteConfig()

	config := aws.GetConfig(githubOwnerFlag)
	awsClient := &aws.Conf

	iamCaller, err := awsClient.GetCallerIdentity()
	if err != nil {
		return err
	}

	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}
	kubefirstStateStoreBucketName := fmt.Sprintf("k1-state-store-%s-%s", clusterNameFlag, clusterId)
	kubefirstArtifactsBucketName := fmt.Sprintf("k1-artifacts-%s-%s", clusterNameFlag, clusterId)

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitStarted, aws.CloudProvider, aws.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	// this branch flag value is overridden with a tag when running from a
	// kubefirst binary for version compatibility
	if gitopsTemplateBranchFlag == "main" && configs.K1Version != "development" {
		gitopsTemplateBranchFlag = configs.K1Version
	}
	log.Info().Msgf("kubefirst version configs.K1Version: %s ", configs.K1Version)
	log.Info().Msgf("cloning gitops-template repo url: %s ", gitopsTemplateURLFlag)
	log.Info().Msgf("cloning gitops-template repo branch: %s ", gitopsTemplateBranchFlag)
	// this branch flag value is overridden with a tag when running from a
	// kubefirst binary for version compatibility
	if metaphorTemplateBranchFlag == "main" && configs.K1Version != "development" {
		metaphorTemplateBranchFlag = configs.K1Version
	}

	log.Info().Msgf("cloning metaphor template url: %s ", metaphorTemplateURLFlag)
	log.Info().Msgf("cloning metaphor template branch: %s ", metaphorTemplateBranchFlag)

	atlantisWebhookSecret := viper.GetString("secrets.atlantis-webhook")
	if atlantisWebhookSecret == "" {
		atlantisWebhookSecret = pkg.Random(20)
		viper.Set("secrets.atlantis-webhook", atlantisWebhookSecret)
		viper.WriteConfig()
	}

	log.Info().Msg("checking authentication to required providers")

	executionControl := viper.GetBool("kubefirst-checks.github-credentials")
	if !executionControl {

		githubToken := os.Getenv("GITHUB_TOKEN")
		if len(githubToken) == 0 {
			return errors.New("please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
		}

		githubWrapper := githubWrapper.New()
		// todo this block need to be pulled into githubHandler. -- begin
		newRepositoryExists := false
		// todo hoist to globals
		newRepositoryNames := []string{"gitops", "metaphor-frontend"}
		errorMsg := "the following repositories must be removed before continuing with your kubefirst installation.\n\t"

		for _, repositoryName := range newRepositoryNames {
			responseStatusCode := githubWrapper.CheckRepoExists(githubOwnerFlag, repositoryName)

			// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository
			repositoryExistsStatusCode := 200
			repositoryDoesNotExistStatusCode := 404

			if responseStatusCode == repositoryExistsStatusCode {
				log.Info().Msgf("repository https://github.com/%s/%s exists", githubOwnerFlag, repositoryName)
				errorMsg = errorMsg + fmt.Sprintf("https://github.com/%s/%s\n\t", githubOwnerFlag, repositoryName)
				newRepositoryExists = true
			} else if responseStatusCode == repositoryDoesNotExistStatusCode {
				log.Info().Msgf("repository https://github.com/%s/%s does not exist, continuing", githubOwnerFlag, repositoryName)
			}
		}
		if newRepositoryExists {
			return errors.New(errorMsg)
		}
		// todo this block need to be pulled into githubHandler. -- end

		// todo this block need to be pulled into githubHandler. -- begin
		newTeamExists := false
		newTeamNames := []string{"admins", "developers"}
		errorMsg = "the following teams must be removed before continuing with your kubefirst installation.\n\t"

		for _, teamName := range newTeamNames {
			responseStatusCode := githubWrapper.CheckTeamExists(githubOwnerFlag, teamName)

			// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#get-a-team-by-name
			teamExistsStatusCode := 200
			teamDoesNotExistStatusCode := 404

			if responseStatusCode == teamExistsStatusCode {
				log.Info().Msgf("team https://github.com/%s/%s exists", githubOwnerFlag, teamName)
				errorMsg = errorMsg + fmt.Sprintf("https://github.com/orgs/%s/teams/%s\n\t", githubOwnerFlag, teamName)
				newTeamExists = true
			} else if responseStatusCode == teamDoesNotExistStatusCode {
				log.Info().Msgf("https://github.com/orgs/%s/teams/%s does not exist, continuing", githubOwnerFlag, teamName)
			}
		}
		if newTeamExists {
			return errors.New(errorMsg)
		}
		// todo this block need to be pulled into githubHandler. -- end
		// todo this should have a collective message of issues for the user
		// todo to clean up with relevant commands

		viper.Set("kubefirst-checks.github-credentials", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed github checks - continuing")
	}

	executionControl = viper.GetBool("kubefirst-checks.cloud-credentials")
	if !executionControl {

		// todo need to verify aws connectivity / credentials have juice
		// also check if creds will expire before the 45 min provision?
		viper.Set("kubefirst-checks.cloud-credentials", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed cloud credentials check - continuing")
	}

	executionControl = viper.GetBool("kubefirst-checks.state-store-create")
	if !executionControl {

		// todo need to create an s3 bucket
		// kubefirst-artifacts-$randomid
		kubefirstStateStoreBucket, err := awsClient.CreateBucket(kubefirstStateStoreBucketName)
		if err != nil {
			return err
		}

		kubefirstArtifactsBucket, err := awsClient.CreateBucket(kubefirstArtifactsBucketName)
		if err != nil {
			return err
		}

		fmt.Println("state store bucket is", strings.ReplaceAll(*kubefirstStateStoreBucket.Location, "/", ""))
		fmt.Println("artifacts bucket is", strings.ReplaceAll(*kubefirstArtifactsBucket.Location, "/", ""))
		// should have argo artifcats and chartmuseum charts
		viper.Set("kubefirst.state-store-bucket", strings.ReplaceAll(*kubefirstStateStoreBucket.Location, "/", ""))
		viper.Set("kubefirst.artifacts-bucket", strings.ReplaceAll(*kubefirstArtifactsBucket.Location, "/", ""))
		viper.Set("kubefirst-checks.state-store-create", true)
		viper.WriteConfig()
		log.Info().Msg("aws s3 buckets created")
	} else {
		log.Info().Msg("already created s3 buckets - continuing")
	}

	skipDomainCheck := viper.GetBool("kubefirst-checks.domain-liveness")
	if !skipDomainCheck {
		// todo check aws domain liveness
		viper.Set("kubefirst-checks.domain-liveness", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("domain check already complete - continuing")
	}

	executionControl = viper.GetBool("kubefirst-checks.kbot-setup")
	if !executionControl {

		log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
		sshPrivateKey, sshPublicKey, err = internalssh.CreateSshKeyPair()
		if err != nil {
			return err
		}
		if len(kbotPasswordFlag) == 0 {
			kbotPasswordFlag = pkg.Random(20)
		}
		log.Info().Msg("ssh key pair creation complete")

		viper.Set("kbot.password", kbotPasswordFlag)
		viper.Set("kbot.private-key", sshPrivateKey)
		viper.Set("kbot.public-key", sshPublicKey)
		viper.Set("kbot.username", "kbot")
		viper.Set("kubefirst-checks.kbot-setup", true)
		viper.WriteConfig()
		log.Info().Msg("kbot-setup complete")
	} else {
		log.Info().Msg("already setup kbot user - continuing")
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitCompleted, aws.CloudProvider, aws.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallStarted, aws.CloudProvider, aws.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	//* generate public keys for ssh
	publicKeys, err := ssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}
	fmt.Println(publicKeys)

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallStarted, aws.CloudProvider, aws.GitProvider); err != nil {
			log.Info().Msg(err.Error())
		}
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := aws.DownloadTools(config)
		if err != nil {
			return err
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}

	atlantisWebhookURL := fmt.Sprintf("https://atlantis.%s/events", domainNameFlag)

	gitopsTemplateTokens := aws.GitOpsDirectoryValues{
		AlertsEmail:                    alertsEmailFlag,
		AwsIamArnAccountRoot:           fmt.Sprintf("arn:aws:iam::%s:root", *iamCaller.Account),
		AwsNodeCapacityType:            "SPOT", // todo adopt cli flag
		ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", domainNameFlag),
		ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", domainNameFlag),
		ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", domainNameFlag),
		ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", domainNameFlag),
		AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", domainNameFlag),
		AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", domainNameFlag),
		AtlantisAllowList:              fmt.Sprintf("github.com/%s/gitops", githubOwnerFlag),
		AtlantisWebhookURL:             atlantisWebhookURL,
		GithubOwner:                    githubOwnerFlag,
		GithubUser:                     githubOwnerFlag,
		GitopsRepoGitURL:               config.DestinationGitopsRepoGitURL,
		DomainName:                     domainNameFlag,
		ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", domainNameFlag),
		CloudProvider:                  aws.CloudProvider,
		CloudRegion:                    cloudRegionFlag,
		ClusterName:                    clusterNameFlag,
		ClusterType:                    clusterTypeFlag,
		GithubHost:                     aws.GithubHost,
		GitDescription:                 "GitHub hosted git",
		GitNamespace:                   "N/A",
		GitProvider:                    civo.GitProvider,
		GitRunner:                      "GitHub Action Runner",
		GitRunnerDescription:           "Self Hosted GitHub Action Runner",
		GitRunnerNS:                    "github-runner",
		GitHubHost:                     "github.com",
		GitHubOwner:                    githubOwnerFlag,
		GitHubUser:                     githubOwnerFlag,
		GitOpsRepoAtlantisWebhookURL:   fmt.Sprintf("https://atlantis.%s/events", domainNameFlag),
		GitOpsRepoGitURL:               config.DestinationGitopsRepoGitURL,
		Kubeconfig:                     config.Kubeconfig,
		KubefirstStateStoreBucket:      kubefirstStateStoreBucketName,
		KubefirstTeam:                  os.Getenv("KUBEFIRST_TEAM"),
		VaultIngressURL:                fmt.Sprintf("https://vault.%s", domainNameFlag),
		MetaphorDevelopmentIngressURL:  fmt.Sprintf("metaphor-development.%s", domainNameFlag),
		MetaphorStagingIngressURL:      fmt.Sprintf("metaphor-staging.%s", domainNameFlag),
		MetaphorProductionIngressURL:   fmt.Sprintf("metaphor-production.%s", domainNameFlag),
		KubefirstVersion:               configs.K1Version,
		VaultIngressNoHTTPSURL:         fmt.Sprintf("metaphor-production.%s", domainNameFlag),
		VouchIngressURL:                fmt.Sprintf("vouch.%s", domainNameFlag),
	}

	fmt.Println("bucket name is ", gitopsTemplateTokens.KubefirstStateStoreBucket)

	//* git clone and detokenize the gitops repository
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		log.Info().Msg("generating your new gitops repository")

		err := aws.PrepareGitopsRepository(clusterNameFlag,
			clusterTypeFlag,
			config.DestinationGitopsRepoGitURL,
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			config.K1Dir,
			&gitopsTemplateTokens,
		)
		if err != nil {
			return err
		}

		// todo emit init telemetry end
		viper.Set("kubefirst-checks.gitops-ready-to-push", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	return errors.New("STOP = checkout the gitops repo locally and see kms key")
	// * create teams and repositories in github
	// todo should terraform-apply-github --> terraform-apply-git-provider
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
	if !executionControl {
		log.Info().Msg("Creating github resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/github"
		tfEnvs := map[string]string{}
		// tfEnvs = aws.GetGithubTerraformEnvs(tfEnvs)
		tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
		tfEnvs["GITHUB_OWNER"] = githubOwnerFlag
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating github resources with terraform %s : %s", tfEntrypoint, err))
		}

		log.Info().Msgf("Created git repositories and teams in github.com/%s", githubOwnerFlag)
		viper.Set("kubefirst-checks.terraform-apply-github", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created github terraform resources")
	}

	//* push detokenized gitops-template repository content to new remote
	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: aws.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", config.DestinationGitopsRepoGitURL)
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/gitops", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("Created git repositories and teams in github.com/%s", githubOwnerFlag)
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
	}

	// todo adopt ecr
	metaphorTemplateTokens := aws.MetaphorTokenValues{}
	metaphorTemplateTokens.ClusterName = clusterNameFlag
	metaphorTemplateTokens.CloudRegion = cloudRegionFlag
	metaphorTemplateTokens.ContainerRegistryURL = fmt.Sprintf("ghcr.io/%s/metaphor-frontend", githubOwnerFlag)
	metaphorTemplateTokens.DomainName = domainNameFlag
	metaphorTemplateTokens.MetaphorDevelopmentIngressURL = fmt.Sprintf("metaphor-development.%s", domainNameFlag)
	metaphorTemplateTokens.MetaphorStagingIngressURL = fmt.Sprintf("metaphor-staging.%s", domainNameFlag)
	metaphorTemplateTokens.MetaphorProductionIngressURL = fmt.Sprintf("metaphor-production.%s", domainNameFlag)

	//* git clone and detokenize the metaphor-frontend-template repository
	if !viper.GetBool("kubefirst-checks.metaphor-repo-pushed") {

		err := aws.PrepareMetaphorRepository(
			config.DestinationMetaphorRepoGitURL,
			config.K1Dir,
			config.MetaphorDir,
			metaphorTemplateBranchFlag,
			metaphorTemplateURLFlag,
			&metaphorTemplateTokens)
		if err != nil {
			return err
		}

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		err = metaphorRepo.Push(&git.PushOptions{
			RemoteName: aws.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			return err
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/metaphor-frontend", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("pushed detokenized metaphor-frontend repository to github.com/%s", githubOwnerFlag)

		viper.Set("kubefirst-checks.metaphor-repo-pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	//* create aws resources
	if !viper.GetBool("kubefirst-checks.terraform-apply-aws") {
		log.Info().Msg("Creating aws cloud resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/aws"
		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_aws_account_id"] = *iamCaller.Account
		tfEnvs["TF_VAR_hosted_zone_name"] = domainNameFlag
		// nodes_graviton := viper.GetBool("aws.nodes_graviton")
		// if nodes_graviton {
		// 	tfEnvs["TF_VAR_ami_type"] = "AL2_ARM_64"
		// 	tfEnvs["TF_VAR_instance_type"] = "t4g.medium"
		// }
		tfEnvs["AWS_SDK_LOAD_CONFIG"] = "1"
		tfEnvs["TF_VAR_aws_region"] = os.Getenv("AWS_REGION")
		tfEnvs["AWS_REGION"] = os.Getenv("AWS_REGION")

		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating aws resources with terraform %s : %s", tfEntrypoint, err))
		}

		log.Info().Msg("Created aws cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-aws", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created aws cluster resources")
	}

	// clientset, err := k8s.GetClientSet(dryRunFlag, config.Kubeconfig)
	// if err != nil {
	// 	return err
	// }

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddawsSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	// executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	// if !executionControl {
	// 	err := aws.AddAwsSecrets(
	// 		atlantisWebhookSecret,
	// 		atlantisWebhookURL,
	// 		viper.GetString("kbot.public-key"),
	// 		config.DestinationGitopsRepoGitURL,
	// 		viper.GetString("kbot.private-key"),
	// 		false,
	// 		githubOwnerFlag,
	// 		config.Kubeconfig,
	// 	)
	// 	if err != nil {
	// 		log.Info().Msg("Error adding kubernetes secrets for bootstrap")
	// 		return err
	// 	}
	// 	viper.Set("kubefirst-checks.k8s-secrets-created", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("already added secrets to aws cluster")
	// }

	// // //* check for ssl restore
	// // log.Info().Msg("checking for tls secrets to restore")
	// // secretsFilesToRestore, err := ioutil.ReadDir(config.SSLBackupDir + "/secrets")
	// // if err != nil {
	// // 	log.Info().Msgf("%s", err)
	// // }
	// // if len(secretsFilesToRestore) != 0 {
	// // 	// todo would like these but requires CRD's and is not currently supported
	// // 	// add crds ( use execShellReturnErrors? )
	// // 	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
	// // 	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
	// // 	// add certificates, and clusterissuers
	// // 	log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
	// // 	ssl.Restore(config.SSLBackupDir, domainNameFlag, config.Kubeconfig)
	// // } else {
	// // 	log.Info().Msg("no files found in secrets directory, continuing")
	// // }

	// //* helm add argo repository && update
	// helmRepo := helm.HelmRepo{
	// 	RepoName:     "argo",
	// 	RepoURL:      "https://argoproj.github.io/argo-helm",
	// 	ChartName:    "argo-cd",
	// 	Namespace:    "argocd",
	// 	ChartVersion: "4.10.5",
	// }

	// //* helm add repo and update
	// executionControl = viper.GetBool("kubefirst-checks.argocd-helm-repo-added")
	// if !executionControl {
	// 	log.Info().Msgf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL)
	// 	helm.AddRepoAndUpdateRepo(dryRunFlag, config.HelmClient, helmRepo, config.Kubeconfig)
	// 	log.Info().Msg("helm repo added")
	// 	viper.Set("kubefirst-checks.argocd-helm-repo-added", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("argo helm repository already added, continuing")
	// }
	// //* helm install argocd
	// executionControl = viper.GetBool("kubefirst-checks.argocd-helm-install")
	// if !executionControl {
	// 	log.Info().Msgf("helm install %s and wait", helmRepo.RepoName)
	// 	// todo adopt golang helm client for helm install
	// 	err := helm.Install(dryRunFlag, config.HelmClient, helmRepo, config.Kubeconfig)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	viper.Set("kubefirst-checks.argocd-helm-install", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("argo helm already installed, continuing")
	// }

	// // Wait for ArgoCD StatefulSet Pods to transition to Running
	// argoCDStatefulSet, err := k8s.ReturnStatefulSetObject(
	// 	config.Kubeconfig,
	// 	"app.kubernetes.io/part-of",
	// 	"argocd",
	// 	"argocd",
	// 	60,
	// )
	// if err != nil {
	// 	log.Info().Msgf("Error finding ArgoCD StatefulSet: %s", err)
	// }
	// _, err = k8s.WaitForStatefulSetReady(config.Kubeconfig, argoCDStatefulSet, 90, false)
	// if err != nil {
	// 	log.Info().Msgf("Error waiting for ArgoCD StatefulSet ready state: %s", err)
	// }

	// //* ArgoCD port-forward
	// argoCDStopChannel := make(chan struct{}, 1)
	// defer func() {
	// 	close(argoCDStopChannel)
	// }()
	// k8s.OpenPortForwardPodWrapper(
	// 	config.Kubeconfig,
	// 	"argocd-server",
	// 	"argocd",
	// 	8080,
	// 	8080,
	// 	argoCDStopChannel,
	// )
	// log.Info().Msgf("port-forward to argocd is available at %s", aws.ArgocdPortForwardURL)

	// var argocdPassword string
	// //* argocd pods are ready, get and set credentials
	// executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	// if !executionControl {
	// 	log.Info().Msg("Setting argocd username and password credentials")

	// 	argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

	// 	argocdPassword = k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
	// 	if argocdPassword == "" {
	// 		log.Info().Msg("argocd password not found in secret")
	// 		return err
	// 	}

	// 	viper.Set("components.argocd.password", argocdPassword)
	// 	viper.Set("components.argocd.username", "admin")
	// 	viper.WriteConfig()
	// 	log.Info().Msg("argocd username and password credentials set successfully")

	// 	log.Info().Msg("Getting an argocd auth token")
	// 	// todo return in here and pass argocdAuthToken as a parameter
	// 	token, err := argocd.GetArgoCDToken("admin", argocdPassword)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Info().Msg("argocd admin auth token set")

	// 	viper.Set("components.argocd.auth-token", token)
	// 	viper.Set("kubefirst-checks.argocd-credentials-set", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("argo credentials already set, continuing")
	// }

	// //* argocd sync registry and start sync waves
	// executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	// if !executionControl {
	// 	log.Info().Msg("applying the registry application to argocd")
	// 	registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", config.K1Dir, clusterNameFlag)
	// 	_, _, err := pkg.ExecShellReturnStrings(config.KubectlClient, "--kubeconfig", config.Kubeconfig, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
	// 	if err != nil {
	// 		log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
	// 		return err
	// 	}
	// 	viper.Set("kubefirst-checks.argocd-create-registry", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("argocd registry create already done, continuing")
	// }

	// // Wait for Vault StatefulSet Pods to transition to Running
	// vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
	// 	config.Kubeconfig,
	// 	"app.kubernetes.io/instance",
	// 	"vault",
	// 	"vault",
	// 	60,
	// )
	// if err != nil {
	// 	log.Info().Msgf("Error finding Vault StatefulSet: %s", err)
	// }
	// _, err = k8s.WaitForStatefulSetReady(config.Kubeconfig, vaultStatefulSet, 60, false)
	// if err != nil {
	// 	log.Info().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
	// }

	// time.Sleep(time.Second * 45)

	// //* vault port-forward
	// vaultStopChannel := make(chan struct{}, 1)
	// defer func() {
	// 	close(vaultStopChannel)
	// }()
	// k8s.OpenPortForwardPodWrapper(
	// 	config.Kubeconfig,
	// 	"vault-0",
	// 	"vault",
	// 	8200,
	// 	8200,
	// 	vaultStopChannel,
	// )

	// //* configure vault with terraform
	// executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	// if !executionControl {
	// 	// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

	// 	//* run vault terraform
	// 	log.Info().Msg("configuring vault with terraform")

	// 	tfEnvs := map[string]string{}

	// 	tfEnvs["TF_VAR_email_address"] = "your@email.com"
	// 	tfEnvs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	// 	tfEnvs["TF_VAR_vault_addr"] = aws.VaultPortForwardURL
	// 	tfEnvs["TF_VAR_vault_token"] = "k1_local_vault_token"
	// 	tfEnvs["VAULT_ADDR"] = aws.VaultPortForwardURL
	// 	tfEnvs["VAULT_TOKEN"] = "k1_local_vault_token"
	// 	tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	// 	tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
	// 	tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
	// 	tfEnvs["TF_VAR_kubefirst_bot_ssh_private_key"] = viper.GetString("kbot.private-key")
	// 	tfEnvs["TF_VAR_aws_access_key_id"] = "kray"
	// 	tfEnvs["TF_VAR_aws_secret_access_key"] = "feedkraystars"

	// 	tfEntrypoint := config.GitopsDir + "/terraform/vault"
	// 	err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Info().Msg("vault terraform executed successfully")
	// 	viper.Set("kubefirst-checks.terraform-apply-vault", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("already executed vault terraform")
	// }

	// //* create users
	// executionControl = viper.GetBool("kubefirst-checks.terraform-apply-users")
	// if !executionControl {
	// 	log.Info().Msg("applying users terraform")

	// 	tfEnvs := map[string]string{}
	// 	tfEnvs["TF_VAR_email_address"] = "your@email.com"
	// 	tfEnvs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	// 	tfEnvs["TF_VAR_vault_addr"] = aws.VaultPortForwardURL
	// 	tfEnvs["TF_VAR_vault_token"] = "k1_local_vault_token"
	// 	tfEnvs["VAULT_ADDR"] = aws.VaultPortForwardURL
	// 	tfEnvs["VAULT_TOKEN"] = "k1_local_vault_token"
	// 	tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	// 	tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
	// 	tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
	// 	tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	// 	tfEnvs["GITHUB_OWNER"] = githubOwnerFlag

	// 	tfEntrypoint := config.GitopsDir + "/terraform/users"
	// 	err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.Info().Msg("executed users terraform successfully")
	// 	// progressPrinter.IncrementTracker("step-users", 1)
	// 	viper.Set("kubefirst-checks.terraform-apply-users", true)
	// 	viper.WriteConfig()
	// } else {
	// 	log.Info().Msg("already created users with terraform")
	// }

	// // Wait for console Deployment Pods to transition to Running
	// consoleDeployment, err := k8s.ReturnDeploymentObject(
	// 	config.Kubeconfig,
	// 	"app.kubernetes.io/instance",
	// 	"kubefirst-console",
	// 	"kubefirst",
	// 	60,
	// )
	// if err != nil {
	// 	log.Info().Msgf("Error finding console Deployment: %s", err)
	// }
	// _, err = k8s.WaitForDeploymentReady(config.Kubeconfig, consoleDeployment, 120)
	// if err != nil {
	// 	log.Info().Msgf("Error waiting for console Deployment ready state: %s", err)
	// }

	// //* console port-forward
	// consoleStopChannel := make(chan struct{}, 1)
	// defer func() {
	// 	close(consoleStopChannel)
	// }()
	// k8s.OpenPortForwardPodWrapper(
	// 	config.Kubeconfig,
	// 	"kubefirst-console",
	// 	"kubefirst",
	// 	8080,
	// 	9094,
	// 	consoleStopChannel,
	// )

	// log.Info().Msg("kubefirst installation complete")
	// log.Info().Msg("welcome to your new kubefirst platform running in aws")

	// err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	// if err != nil {
	// 	log.Error().Err(err).Msg("")
	// }

	// err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	// if err != nil {
	// 	log.Error().Err(err).Msg("")
	// }

	// reports.LocalHandoffScreenV2(argocdPassword, clusterNameFlag, githubOwnerFlag, config, dryRunFlag, false)

	// if useTelemetryFlag {
	// 	if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallCompleted, aws.CloudProvider, aws.GitProvider); err != nil {
	// 		log.Info().Msg(err.Error())
	// 		return err
	// 	}
	// }

	time.Sleep(time.Millisecond * 100) // allows progress bars to finish

	return nil
}
