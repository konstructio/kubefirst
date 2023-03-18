package aws

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	awsinternal "github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/bootstrap"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/github"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/segment"
	"github.com/kubefirst/kubefirst/internal/services"
	internalssh "github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if strings.Contains(domainNameFlag, "://") {
		return errors.New("--domain-name should not have a protocol, please provide it in the form of \n\tdomain.com or my.domain.com")
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

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		return err
	}

	kbotPasswordFlag, err := cmd.Flags().GetString("kbot-password")
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
	githubUser, err := gitHubHandler.GetGitHubUser(os.Getenv("GITHUB_TOKEN"))
	if err != nil {
		return err
	}

	// required for destroy command
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.dry-run", dryRunFlag)
	viper.Set("flags.github-owner", githubOwnerFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.WriteConfig()

	config := awsinternal.GetConfig(githubOwnerFlag)
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClientNoSSL := http.Client{Transport: customTransport}
	awsClient := &awsinternal.Conf
	segmentClient := &segment.Client
	vaultClient := &vault.Conf

	var (
		kubefirstStateStoreBucketName string
		kubefirstArtifactsBucketName  string
		vaultRootToken                string
	)

	iamCaller, err := awsClient.GetCallerIdentity()
	if err != nil {
		return err
	}
	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}
	kubefirstStateStoreBucketName = fmt.Sprintf("k1-state-store-%s-%s", clusterNameFlag, clusterId)
	kubefirstArtifactsBucketName = fmt.Sprintf("k1-artifacts-%s-%s", clusterNameFlag, clusterId)

	if useTelemetryFlag {
		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, awsinternal.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricInitStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
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

		githubSession := github.New(os.Getenv("GITHUB_TOKEN"))
		// todo this block need to be pulled into githubHandler. -- begin
		newRepositoryExists := false
		// todo hoist to globals
		newRepositoryNames := []string{"gitops", "metaphor"}
		errorMsg := "the following repositories must be removed before continuing with your kubefirst installation.\n\t"

		for _, repositoryName := range newRepositoryNames {
			responseStatusCode := githubSession.CheckRepoExists(githubOwnerFlag, repositoryName)

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
			responseStatusCode := githubSession.CheckTeamExists(githubOwnerFlag, teamName)

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
		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, awsinternal.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricInitCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, awsinternal.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricMgmtClusterInstallStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	}

	//* generate public keys for ssh
	publicKeys, err := ssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}
	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := awsinternal.DownloadTools(config)
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
	awsAccountID := *iamCaller.Account

	gitopsTemplateTokens := awsinternal.GitOpsDirectoryValues{
		AlertsEmail:                    alertsEmailFlag,
		AwsIamArnAccountRoot:           fmt.Sprintf("arn:aws:iam::%s:root", *iamCaller.Account),
		AwsNodeCapacityType:            "ON_DEMAND", // todo adopt cli flag
		AwsAccountID:                   *iamCaller.Account,
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
		CloudProvider:                  awsinternal.CloudProvider,
		CloudRegion:                    cloudRegionFlag,
		ClusterName:                    clusterNameFlag,
		ClusterType:                    clusterTypeFlag,
		GithubHost:                     awsinternal.GithubHost,
		GitDescription:                 "GitHub hosted git",
		GitNamespace:                   "N/A",
		GitProvider:                    awsinternal.GitProvider,
		GitRunner:                      "GitHub Action Runner",
		GitRunnerDescription:           "Self Hosted GitHub Action Runner",
		GitRunnerNS:                    "github-runner",
		GitHubHost:                     "github.com",
		GitHubOwner:                    githubOwnerFlag,
		GitHubUser:                     githubUser,
		GitOpsRepoAtlantisWebhookURL:   fmt.Sprintf("https://atlantis.%s/events", domainNameFlag),
		GitOpsRepoGitURL:               config.DestinationGitopsRepoGitURL,
		Kubeconfig:                     config.Kubeconfig,
		KubefirstArtifactsBucket:       kubefirstArtifactsBucketName,
		KubefirstStateStoreBucket:      kubefirstStateStoreBucketName,
		KubefirstTeam:                  os.Getenv("KUBEFIRST_TEAM"),
		VaultIngressURL:                fmt.Sprintf("https://vault.%s", domainNameFlag),
		MetaphorDevelopmentIngressURL:  fmt.Sprintf("metaphor-development.%s", domainNameFlag),
		MetaphorStagingIngressURL:      fmt.Sprintf("metaphor-staging.%s", domainNameFlag),
		MetaphorProductionIngressURL:   fmt.Sprintf("metaphor-production.%s", domainNameFlag),
		KubefirstVersion:               configs.K1Version,
		VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", domainNameFlag),
		VouchIngressURL:                fmt.Sprintf("vouch.%s", domainNameFlag),
	}

	metaphorTemplateTokens := awsinternal.MetaphorTokenValues{
		ClusterName:                   clusterNameFlag,
		CloudRegion:                   cloudRegionFlag,
		ContainerRegistryURL:          fmt.Sprintf("ghcr.io/%s/metaphor", githubOwnerFlag),
		DomainName:                    domainNameFlag,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", domainNameFlag),
		MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", domainNameFlag),
		MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", domainNameFlag),
	}

	//* git clone and detokenize the gitops repository
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		log.Info().Msg("generating your new gitops repository")

		err := awsinternal.PrepareGitRepositories(
			gitProviderFlag,
			clusterNameFlag,
			clusterTypeFlag,
			config.DestinationGitopsRepoGitURL,
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			config.DestinationMetaphorRepoGitURL,
			config.K1Dir,
			&gitopsTemplateTokens,
			config.MetaphorDir,
			&metaphorTemplateTokens,
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

	// * create teams and repositories in github
	// todo should terraform-apply-github --> terraform-apply-git-provider
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
	if !executionControl {
		log.Info().Msg("Creating github resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/github"
		tfEnvs := map[string]string{}
		// tfEnvs = awsinternal.GetGithubTerraformEnvs(tfEnvs)
		tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
		tfEnvs["GITHUB_OWNER"] = githubOwnerFlag
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return fmt.Errorf("error creating github resources with terraform %s : %s", tfEntrypoint, err)
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
			RemoteName: awsinternal.GitProvider,
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

	//* git clone and detokenize the metaphor-template repository
	if !viper.GetBool("kubefirst-checks.metaphor-repo-pushed") {

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		err = metaphorRepo.Push(&git.PushOptions{
			RemoteName: awsinternal.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			return err
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/metaphor", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("pushed detokenized metaphor repository to github.com/%s", githubOwnerFlag)

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
		tfEnvs["TF_VAR_aws_account_id"] = awsAccountID
		tfEnvs["TF_VAR_hosted_zone_name"] = domainNameFlag
		tfEnvs["AWS_SDK_LOAD_CONFIG"] = "1"
		tfEnvs["TF_VAR_aws_region"] = os.Getenv("AWS_REGION")
		tfEnvs["AWS_REGION"] = os.Getenv("AWS_REGION")

		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return fmt.Errorf("error creating aws resources with terraform %s : %s", tfEntrypoint, err)
		}

		log.Info().Msg("Created aws cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-aws", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created aws cluster resources")
	}

	//* create aws resources
	if !viper.GetBool("kubefirst-checks.detokenize-kms") {
		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			return err
		}
		awsKmsKeyId, err := awsClient.GetKmsKeyID(fmt.Sprintf("alias/vault_%s", clusterNameFlag))
		if err != nil {
			return err
		}
		gitopsTemplateTokens.AwsKmsKeyId = awsKmsKeyId

		if err := pkg.ReplaceFileContent(
			fmt.Sprintf("%s/registry/%s/components/vault/application.yaml", config.GitopsDir, clusterNameFlag),
			"<AWS_KMS_KEY_ID>",
			gitopsTemplateTokens.AwsKmsKeyId,
		); err != nil {
			return err
		}

		err = gitClient.Commit(gitopsRepo, "committing detoknized kms key")
		if err != nil {
			return err
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: awsinternal.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			return err
		}
		log.Info().Msg("pushed detokenized kms key to gitops")
		viper.Set("kubefirst-checks.detokenize-kms", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already pushed kms key to gitops")
	}

	// todo create a client from config!! need to re-use AwsConfiguration or adopt session in the other direction
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cloudRegionFlag),
	}))

	eksSvc := eks.New(sess)

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterNameFlag),
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

	argocdClient, err := argocdapi.NewForConfig(restConfig)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// //* check for ssl restore
	// log.Info().Msg("checking for tls secrets to restore")
	// secretsFilesToRestore, err := ioutil.ReadDir(config.SSLBackupDir + "/secrets")
	// if err != nil {
	// 	log.Info().Msgf("%s", err)
	// }
	// if len(secretsFilesToRestore) != 0 {
	// 	// todo would like these but requires CRD's and is not currently supported
	// 	// add crds ( use execShellReturnErrors? )
	// 	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
	// 	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
	// 	// add certificates, and clusterissuers
	// 	log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
	// 	ssl.Restore(config.SSLBackupDir, domainNameFlag, config.Kubeconfig)
	// } else {
	// 	log.Info().Msg("no files found in secrets directory, continuing")
	// }

	//* install argocd
	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
		log.Info().Msgf("installing argocd")
		err = argocd.ApplyArgoCDKustomize(clientset)
		if err != nil {
			return err
		}
		viper.Set("kubefirst-checks.argocd-install", true)
		viper.WriteConfig()
		// progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argo cd already installed, continuing")
		// progressPrinter.IncrementTracker("installing-argo-cd", 1)
	}

	// Wait for ArgoCD to be ready
	_, err = k8s.VerifyArgoCDReadiness(clientset, true)
	if err != nil {
		log.Fatal().Msgf("error waiting for ArgoCD to become ready: %s", err)
	}

	//* ArgoCD port-forward
	// todo DO WE ACTUALLY USE THIS!?
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
	log.Info().Msgf("port-forward to argocd is available at %s", awsinternal.ArgocdPortForwardURL)

	// todo need to create argocd repo secret in the cluster
	//* create argocd kubernetes secret for connectivity to private gitops repo
	executionControl = viper.GetBool("kubefirst-checks.bootstrap-cluster")
	if !executionControl {
		log.Info().Msg("Setting argocd username and password credentials")

		log.Info().Msg("creating service accounts and namespaces")
		err = bootstrap.ServiceAccounts(clientset)
		if err != nil {
			return err
		}

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "repo-credentials-template",
				Namespace:   "argocd",
				Annotations: map[string]string{"managed-by": "argocd.argoproj.io"},
				Labels:      map[string]string{"argocd.argoproj.io/secret-type": "repository"},
			},
			Data: map[string][]byte{
				"type":          []byte("git"),
				"name":          []byte(fmt.Sprintf("%s-gitops", githubOwnerFlag)),
				"url":           []byte(config.DestinationGitopsRepoGitURL),
				"sshPrivateKey": []byte(viper.GetString("kbot.private-key")),
			},
		}

		_, err := clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Get(context.TODO(), secret.ObjectMeta.Name, metav1.GetOptions{})
		if err == nil {
			log.Info().Msgf("kubernetes secret %s/%s already created - skipping", secret.Namespace, secret.Name)
		} else if strings.Contains(err.Error(), "not found") {
			err := k8s.CreateSecretV2(clientset, secret)
			if err != nil {
				log.Info().Msgf("error creating kubernetes secret %s/%s: %s", secret.Namespace, secret.Name, err)
				return err
			}
			log.Info().Msgf("created kubernetes secret: %s/%s", secret.Namespace, secret.Name)
		}

		log.Info().Msg("secret create for argocd to connect to gitops repo")

		ecrToken, err := awsClient.GetECRAuthToken()
		if err != nil {
			return err
		}

		usernamePasswordString := fmt.Sprintf("%s:%s", pkg.AwsECRUsername, ecrToken)
		usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		registryURL := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", awsAccountID, cloudRegionFlag)

		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, registryURL, usernamePasswordStringB64)

		dockerCfgSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "argo"},
			Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
			Type:       "Opaque",
		}
		_, err = clientset.CoreV1().Secrets(dockerCfgSecret.ObjectMeta.Namespace).Create(context.TODO(), dockerCfgSecret, metav1.CreateOptions{})
		if err != nil {
			log.Info().Msgf("error creating kubernetes secret %s/%s: %s", dockerCfgSecret.Namespace, dockerCfgSecret.Name, err)
			return err
		}

		viper.Set("kubefirst-checks.bootstrap-cluster", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("argo credentials already set, continuing")
	}

	var argocdPassword string
	//* argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	if !executionControl {
		log.Info().Msg("Setting argocd username and password credentials")

		secData, err := k8s.ReadSecretV2(clientset, "argocd", "argocd-initial-admin-secret")
		if err != nil {
			return err
		}
		argocdPassword = secData["password"]

		viper.Set("components.argocd.password", argocdPassword)
		viper.Set("components.argocd.username", "admin")
		viper.WriteConfig()
		log.Info().Msg("argocd username and password credentials set successfully")

		log.Info().Msg("Getting an argocd auth token")
		// todo return in here and pass argocdAuthToken as a parameter
		token, err := argocd.GetArgocdTokenV2(&httpClientNoSSL, pkg.ArgocdPortForwardURL, "admin", argocdPassword)
		if err != nil {
			return err
		}

		log.Info().Msg("argocd admin auth token set")

		viper.Set("components.argocd.auth-token", token)
		viper.Set("kubefirst-checks.argocd-credentials-set", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("argo credentials already set, continuing")
	}

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		log.Info().Msg("applying the registry application to argocd")
		registryApplicationObject, err := argocd.GetArgoCDApplicationObject(config.DestinationGitopsRepoGitURL, fmt.Sprintf("registry/%s", clusterNameFlag))
		if err != nil {
			return err
		}

		_, err = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			return err
		}
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
	}

	//* helm install argocd
	executionControl = viper.GetBool("kubefirst-checks.vault-ready")
	if !executionControl {
		log.Info().Msg("waiting for vault pods to be ready ")
		// Wait for Vault StatefulSet Pods to transition to Running
		vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
			clientset,
			"app.kubernetes.io/instance",
			"vault",
			"vault",
			60,
		)
		if err != nil {
			log.Info().Msgf("Error finding Vault StatefulSet: %s", err)
		}
		_, err = k8s.WaitForStatefulSetReady(clientset, vaultStatefulSet, 60, false)
		if err != nil {
			log.Info().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
		}

		time.Sleep(time.Second * 20) // todo remove this? might not be needed anymore
		viper.Set("kubefirst-checks.vault-ready", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("vault is ready, continuing")
	}

	//* vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		clientset,
		restConfig,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.vault-unseal")
	if !executionControl {
		log.Info().Msg("insealing vault unseal")

		initResponse, err := vaultClient.AutoUnseal()
		if err != nil {
			return err
		}

		vaultRootToken = initResponse.RootToken

		dataToWrite := make(map[string][]byte)
		dataToWrite["root-token"] = []byte(vaultRootToken)
		for i, value := range initResponse.Keys {
			dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
		}
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vault.VaultSecretName,
				Namespace: vault.VaultNamespace,
			},
			Data: dataToWrite,
		}

		err = k8s.CreateSecretV2(clientset, &secret)
		if err != nil {
			return err
		}

		viper.Set("kubefirst-checks.vault-unseal", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("vault unseal already done, continuing")
	}

	vaultRootTokenLookup, err := k8s.ReadSecretV2(clientset, "vault", "vault-unseal-secret")
	if err != nil {
		return err
	}

	vaultRootToken = vaultRootTokenLookup["root-token"]

	//* configure vault with terraform
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

		//* run vault terraform
		log.Info().Msg("configuring vault with terraform")

		tfEnvs := map[string]string{}

		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
		tfEnvs["TF_VAR_vault_addr"] = awsinternal.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = awsinternal.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = atlantisWebhookSecret
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
		tfEnvs["TF_VAR_kubefirst_bot_ssh_private_key"] = viper.GetString("kbot.private-key") // todo hyrdate a variable up top with these so we dont ref viper.

		tfEntrypoint := config.GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}

		log.Info().Msg("vault terraform executed successfully")
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already executed vault terraform")
	}

	//* create users
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-users")
	if !executionControl {
		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
		tfEnvs["TF_VAR_vault_addr"] = awsinternal.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = awsinternal.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
		tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
		tfEnvs["GITHUB_OWNER"] = githubOwnerFlag

		tfEntrypoint := config.GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}
		log.Info().Msg("executed users terraform successfully")
		// progressPrinter.IncrementTracker("step-users", 1)
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created users with terraform")
	}

	// Wait for console Deployment Pods to transition to Running
	consoleDeployment, err := k8s.ReturnDeploymentObject(
		clientset,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding console Deployment: %s", err)
	}
	_, err = k8s.WaitForDeploymentReady(clientset, consoleDeployment, 120)
	if err != nil {
		log.Info().Msgf("Error waiting for console Deployment ready state: %s", err)
	}

	//* console port-forward
	consoleStopChannel := make(chan struct{}, 1)
	defer func() {
		close(consoleStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		clientset,
		restConfig,
		"kubefirst-console",
		"kubefirst",
		8080,
		9094,
		consoleStopChannel,
	)

	log.Info().Msg("kubefirst installation complete")
	log.Info().Msg("welcome to your new kubefirst platform running in aws")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	//! reports.LocalHandoffScreenV2(argocdPassword, clusterNameFlag, githubOwnerFlag, config, dryRunFlag, false)

	if useTelemetryFlag {
		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, awsinternal.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricMgmtClusterInstallCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	}

	time.Sleep(time.Millisecond * 100) // allows progress bars to finish

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return nil
}
