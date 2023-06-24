/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/atotto/clipboard"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/internal/gitShim"
	"github.com/kubefirst/kubefirst/internal/telemetryShim"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/bootstrap"
	"github.com/kubefirst/runtime/pkg/dns"
	"github.com/kubefirst/runtime/pkg/gitClient"
	"github.com/kubefirst/runtime/pkg/github"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/handlers"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/progressPrinter"
	"github.com/kubefirst/runtime/pkg/reports"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/services"
	internalssh "github.com/kubefirst/runtime/pkg/ssh"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/kubefirst/runtime/pkg/vault"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createAws(cmd *cobra.Command, args []string) error {
	alertsEmailFlag, err := cmd.Flags().GetString("alerts-email")
	if err != nil {
		return err
	}

	ciFlag, err := cmd.Flags().GetBool("ci")
	if err != nil {
		return err
	}

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

	githubOrgFlag, err := cmd.Flags().GetString("github-org")
	if err != nil {
		return err
	}

	gitlabGroupFlag, err := cmd.Flags().GetString("gitlab-group")
	if err != nil {
		return err
	}

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
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

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return err
	}

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		return fmt.Errorf("this cluster install process has already completed successfully")
	}

	utilities.CreateK1ClusterDirectory(clusterNameFlag)
	helpers.DisplayLogHints()

	switch gitProviderFlag {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "github.com", key.Type())
		}
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "gitlab.com", key.Type())
		}
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(8080, 8200, 9094)
	if err != nil {
		return fmt.Errorf("%s - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing", err.Error())
	}

	// Validate aws region
	awsClient := &awsinternal.AWSConfiguration{
		Config: awsinternal.NewAwsV2(cloudRegionFlag),
	}

	_, err = awsClient.CheckAvailabilityZones(cloudRegionFlag)
	if err != nil {
		return err
	}

	// required for destroy command
	viper.Set("flags.alerts-email", alertsEmailFlag)
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.cloud-region", cloudRegionFlag)
	viper.WriteConfig()

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// Switch based on git provider, set params
	var cGitHost, cGitOwner, cGitToken, cGitUser string
	var cGitlabOwnerGroupID int
	switch gitProviderFlag {
	case "github":
		if githubOrgFlag == "" {
			return fmt.Errorf("please provide a github organization using the --github-org flag")
		}
		if os.Getenv("GITHUB_TOKEN") == "" {
			return fmt.Errorf("your GITHUB_TOKEN is not set. Please set and try again")
		}

		cGitHost = awsinternal.GithubHost
		cGitOwner = githubOrgFlag
		cGitToken = os.Getenv("GITHUB_TOKEN")

		// Verify token scopes
		err = github.VerifyTokenPermissions(cGitToken)
		if err != nil {
			return err
		}

		// Handle authorization checks
		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		// get github data to set user based on the provided token
		log.Info().Msg("verifying github authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(cGitToken)
		if err != nil {
			return err
		}
		cGitUser = githubUser
		viper.Set("github.user", githubUser)
		err = viper.WriteConfig()
		if err != nil {
			return err
		}
		err = gitHubHandler.CheckGithubOrganizationPermissions(cGitToken, githubOrgFlag, githubUser)
		if err != nil {
			return err
		}
		viper.Set("flags.github-owner", githubOrgFlag)
		viper.WriteConfig()
	case "gitlab":
		if gitlabGroupFlag == "" {
			return fmt.Errorf("please provide a gitlab group using the --gitlab-group flag")
		}
		if os.Getenv("GITLAB_TOKEN") == "" {
			return fmt.Errorf("your GITLAB_TOKEN is not set. please set and try again")
		}

		cGitToken = os.Getenv("GITLAB_TOKEN")

		// Verify token scopes
		err = gitlab.VerifyTokenPermissions(cGitToken)
		if err != nil {
			return err
		}

		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
		if err != nil {
			return err
		}

		cGitHost = awsinternal.GitlabHost
		cGitOwner = gitlabClient.ParentGroupPath
		cGitlabOwnerGroupID = gitlabClient.ParentGroupID
		log.Info().Msgf("set gitlab owner to %s", cGitOwner)

		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		cGitUser = user.Username

		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
		viper.Set("flags.gitlab-owner-group-id", cGitlabOwnerGroupID)
		viper.WriteConfig()
	default:
		log.Error().Msgf("invalid git provider option")
	}

	// Instantiate config
	config := awsinternal.GetConfig(clusterNameFlag, domainNameFlag, gitProviderFlag, cGitOwner)

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClientNoSSL := http.Client{Transport: customTransport}

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

	// Segment Client
	segmentClient := &segment.SegmentClient{
		CliVersion:        configs.K1Version,
		CloudProvider:     awsinternal.CloudProvider,
		ClusterID:         clusterId,
		ClusterType:       clusterTypeFlag,
		DomainName:        domainNameFlag,
		GitProvider:       gitProviderFlag,
		KubefirstClient:   "cli",
		KubefirstTeam:     kubefirstTeam,
		KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
	}
	segmentClient.SetupClient()
	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)
	telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricInitStarted, "")

	// this branch flag value is overridden with a tag when running from a
	// kubefirst binary for version compatibility
	switch configs.K1Version {
	case "development":
		if strings.Contains(gitopsTemplateURLFlag, "https://github.com/kubefirst/gitops-template.git") && gitopsTemplateBranchFlag == "" {
			gitopsTemplateBranchFlag = "main"
		}
	default:
		switch gitopsTemplateURLFlag {
		case "https://github.com/kubefirst/gitops-template.git":
			if gitopsTemplateBranchFlag == "" {
				gitopsTemplateBranchFlag = configs.K1Version
			}
		default:
			if gitopsTemplateBranchFlag == "" {
				return fmt.Errorf("must supply gitops-template-branch flag when gitops-template-url is overridden")
			}
		}
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

	// Objects to check for
	newRepositoryNames := []string{"gitops", "metaphor"}
	newTeamNames := []string{"admins", "developers"}

	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider))
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitCredentialsCheckStarted, "")
		if len(cGitToken) == 0 {
			msg := fmt.Sprintf(
				"please set a %s_TOKEN environment variable to continue",
				strings.ToUpper(config.GitProvider),
			)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitCredentialsCheckFailed, msg)
			return fmt.Errorf(msg)
		}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  gitProviderFlag,
			GitToken:     cGitToken,
			GitOwner:     cGitOwner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}
		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			return err
		}

		viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider), true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitCredentialsCheckCompleted, "")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg(fmt.Sprintf("already completed %s checks - continuing", config.GitProvider))
		progressPrinter.IncrementTracker("preflight-checks", 1)
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
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateStarted, "")

		//
		kubefirstStateStoreBucket, err := awsClient.CreateBucket(kubefirstStateStoreBucketName)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
			return err
		}

		kubefirstArtifactsBucket, err := awsClient.CreateBucket(kubefirstArtifactsBucketName)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
			return err
		}

		log.Info().Msgf("state store bucket is %s", *kubefirstStateStoreBucket.Location)
		log.Info().Msgf("artifacts bucket is %s", *kubefirstArtifactsBucket.Location)

		viper.Set("kubefirst.state-store-bucket", strings.ReplaceAll(*kubefirstStateStoreBucket.Location, "/", ""))
		viper.Set("kubefirst.artifacts-bucket", strings.ReplaceAll(*kubefirstArtifactsBucket.Location, "/", ""))
		viper.Set("kubefirst-checks.state-store-create", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateCompleted, "")
		log.Info().Msg("aws s3 buckets created")
	} else {
		log.Info().Msg("already created s3 buckets - continuing")
	}

	skipDomainCheck := viper.GetBool("kubefirst-checks.domain-liveness")
	if !skipDomainCheck {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessStarted, "")

		// verify dns
		isPrivateZone, nameServers, err := awsClient.GetHostedZoneNameServers(domainNameFlag)
		if err != nil {
			return err
		}

		if !isPrivateZone {
			err = dns.VerifyProviderDNS("aws", cloudRegionFlag, domainNameFlag, nameServers)
			if err != nil {
				return err
			}
		}

		domainLiveness := awsClient.TestHostedZoneLiveness(domainNameFlag)
		if !domainLiveness {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessFailed, "domain liveness test failed")
			msg := "failed to check the liveness of the HostedZone. A valid public HostedZone on the same AWS " +
				"account as the one where Kubefirst will be installed is required for this operation to " +
				"complete.\nTroubleshoot Steps:\n\n - Make sure you are using the correct AWS account and " +
				"region.\n - Verify that you have the necessary permissions to access the hosted zone.\n - Check " +
				"that the hosted zone is correctly configured and is a public hosted zone\n - Check if the " +
				"hosted zone exists and has the correct name and domain.\n - If you don't have a HostedZone," +
				"please follow these instructions to create one: " +
				"https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html \n\n" +
				"if you are still facing issues please reach out to support team for further assistance"

			return fmt.Errorf(msg)
		}
		viper.Set("kubefirst-checks.domain-liveness", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessCompleted, "")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("domain check already complete - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	executionControl = viper.GetBool("kubefirst-checks.kbot-setup")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricKbotSetupStarted, "")

		log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
		sshPrivateKey, sshPublicKey, err = internalssh.CreateSshKeyPair()
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricKbotSetupFailed, err.Error())
			return err
		}
		log.Info().Msg("ssh key pair creation complete")

		viper.Set("kbot.private-key", sshPrivateKey)
		viper.Set("kbot.public-key", sshPublicKey)
		viper.Set("kbot.username", "kbot")
		viper.Set("kubefirst-checks.kbot-setup", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricKbotSetupCompleted, "")
		log.Info().Msg("kbot-setup complete")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already setup kbot user - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricInitCompleted, "")
	telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricClusterInstallStarted, "")

	publicKeys, err := ssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := awsinternal.DownloadTools(config, awsinternal.KubectlClientVersion, awsinternal.TerraformClientVersion)
		if err != nil {
			return err
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	atlantisWebhookURL := fmt.Sprintf("https://atlantis.%s/events", domainNameFlag)
	awsAccountID := *iamCaller.Account
	registryURL := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", awsAccountID, cloudRegionFlag)

	gitopsTemplateTokens := awsinternal.GitOpsDirectoryValues{
		AlertsEmail:               alertsEmailFlag,
		AtlantisAllowList:         fmt.Sprintf("%s/%s/*", cGitHost, cGitOwner),
		AwsIamArnAccountRoot:      fmt.Sprintf("arn:aws:iam::%s:root", *iamCaller.Account),
		AwsNodeCapacityType:       "ON_DEMAND", // todo adopt cli flag
		AwsAccountID:              *iamCaller.Account,
		CloudProvider:             awsinternal.CloudProvider,
		CloudRegion:               cloudRegionFlag,
		ClusterName:               clusterNameFlag,
		ClusterType:               clusterTypeFlag,
		DomainName:                domainNameFlag,
		Kubeconfig:                config.Kubeconfig,
		KubefirstArtifactsBucket:  kubefirstArtifactsBucketName,
		KubefirstStateStoreBucket: kubefirstStateStoreBucketName,
		KubefirstTeam:             os.Getenv("KUBEFIRST_TEAM"),
		KubefirstVersion:          configs.K1Version,

		ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", domainNameFlag),
		ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", domainNameFlag),
		ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", domainNameFlag),
		ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", domainNameFlag),
		AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", domainNameFlag),
		AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", domainNameFlag),
		ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", domainNameFlag),
		VaultIngressURL:                fmt.Sprintf("https://vault.%s", domainNameFlag),
		VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", domainNameFlag),
		VouchIngressURL:                fmt.Sprintf("https://vouch.%s", domainNameFlag),

		GitDescription:       fmt.Sprintf("%s hosted git", config.GitProvider),
		GitNamespace:         "N/A",
		GitProvider:          config.GitProvider,
		GitRunner:            fmt.Sprintf("%s Runner", config.GitProvider),
		GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", config.GitProvider),
		GitRunnerNS:          fmt.Sprintf("%s-runner", config.GitProvider),
		GitURL:               gitopsTemplateURLFlag,

		GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", cGitOwner),
		GitHubOwner: cGitOwner,
		GitHubUser:  cGitUser,

		GitlabHost:         awsinternal.GitlabHost,
		GitlabOwner:        cGitOwner,
		GitlabOwnerGroupID: viper.GetInt("flags.gitlab-owner-group-id"),
		GitlabUser:         cGitUser,

		GitOpsRepoAtlantisWebhookURL: fmt.Sprintf("https://atlantis.%s/events", domainNameFlag),
		GitOpsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", cGitHost, cGitOwner),
		ClusterId:                    clusterId,

		AtlantisWebhookURL:   atlantisWebhookURL,
		ContainerRegistryURL: registryURL,
	}

	metaphorTemplateTokens := awsinternal.MetaphorTokenValues{
		ClusterName:                   clusterNameFlag,
		CloudRegion:                   cloudRegionFlag,
		ContainerRegistryURL:          registryURL,
		DomainName:                    domainNameFlag,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", domainNameFlag),
		MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", domainNameFlag),
		MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", domainNameFlag),
	}

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {
	var destinationGitopsRepoGitURL, destinationMetaphorRepoGitURL string

	progressPrinter.AddTracker("cloning-and-formatting-git-repositories", "Cloning and formatting git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		log.Info().Msg("generating your new gitops repository")

		// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
		switch config.GitProvider {
		case "github":
			destinationGitopsRepoGitURL = config.DestinationGitopsRepoGitURL
			destinationMetaphorRepoGitURL = config.DestinationMetaphorRepoGitURL
		case "gitlab":
			gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
			if err != nil {
				return err
			}
			// Format git url based on full path to group
			destinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
			destinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)

		}
		// These need to be set for reference elsewhere
		viper.Set(fmt.Sprintf("%s.repos.gitops.git-url", config.GitProvider), destinationGitopsRepoGitURL)
		viper.WriteConfig()
		gitopsTemplateTokens.GitOpsRepoGitURL = destinationGitopsRepoGitURL

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
		progressPrinter.IncrementTracker("cloning-and-formatting-git-repositories", 1)
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
		progressPrinter.IncrementTracker("cloning-and-formatting-git-repositories", 1)
	}

	// * handle git terraform apply
	progressPrinter.AddTracker("applying-git-terraform", fmt.Sprintf("Applying %s Terraform", config.GitProvider), 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	switch config.GitProvider {
	case "github":
		// //* create teams and repositories in github
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
		if !executionControl {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

			log.Info().Msg("Creating github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			// tfEnvs = awsinternal.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITHUB_TOKEN"] = cGitToken
			tfEnvs["GITHUB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
				telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info().Msgf("Created git repositories and teams for github.com/%s", cGitOwner)
			viper.Set("kubefirst-checks.terraform-apply-github", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		} else {
			log.Info().Msg("already created github terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	case "gitlab":
		// //* create teams and repositories in gitlab
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-gitlab")
		if !executionControl {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

			log.Info().Msg("Creating gitlab resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			// tfEnvs = awsinternal.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GITLAB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(viper.GetInt("flags.gitlab-owner-group-id"))
			tfEnvs["TF_VAR_gitlab_owner"] = viper.GetString("flags.gitlab-owner")
			err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
				telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyFailed, msg)
				return fmt.Errorf(msg)
			}

			log.Info().Msgf("created git projects and groups for gitlab.com/%s", gitlabGroupFlag)
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
			viper.Set("kubefirst-checks.terraform-apply-gitlab", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyCompleted, "")
		} else {
			log.Info().Msg("already created gitlab terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	}

	// * push detokenized gitops-template repository content to new remote
	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %s", destinationGitopsRepoGitURL)
	log.Info().Msgf("referencing metaphor repository: %s", destinationMetaphorRepoGitURL)

	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushStarted, "")

		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		err = internalssh.EvalSSHKey(&internalssh.EvalSSHKeyRequest{
			GitProvider:     gitProviderFlag,
			GitlabGroupFlag: gitlabGroupFlag,
			GitToken:        cGitToken,
		})
		if err != nil {
			return err
		}

		// Push gitops repo to remote
		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: config.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", destinationGitopsRepoGitURL, err)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			log.Panic().Msg(msg)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       publicKeys,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized metaphor repository to remote %s: %s", destinationMetaphorRepoGitURL, err)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			log.Panic().Msg(msg)
		}

		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("successfully pushed gitops to git@%s/%s/gitops", cGitHost, cGitOwner)
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushCompleted, "")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)
	}

	//* create aws resources
	progressPrinter.AddTracker("applying-aws-terraform", "Applying AWS Terraform", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.terraform-apply-aws") {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

		log.Info().Msg("Creating aws cloud resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/aws"
		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_aws_account_id"] = awsAccountID
		tfEnvs["TF_VAR_hosted_zone_name"] = domainNameFlag
		tfEnvs["AWS_SDK_LOAD_CONFIG"] = "1"
		tfEnvs["TF_VAR_aws_region"] = cloudRegionFlag
		tfEnvs["AWS_REGION"] = cloudRegionFlag

		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			msg := fmt.Sprintf("error creating aws resources with terraform %s : %s", tfEntrypoint, err)
			viper.Set("kubefirst-checks.terraform-apply-aws-failed", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
			return fmt.Errorf(msg)
		}

		log.Info().Msg("Created aws cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-aws", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("applying-aws-terraform", 1)
	} else {
		log.Info().Msg("already created aws cluster resources")
		progressPrinter.IncrementTracker("applying-aws-terraform", 1)
	}

	progressPrinter.AddTracker("applying-kms", "Applying AWS KMS", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
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

		err = gitClient.Commit(gitopsRepo, "committing detokenized kms key")
		if err != nil {
			return err
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: config.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			return err
		}
		log.Info().Msg("pushed detokenized kms key to gitops")
		viper.Set("kubefirst-checks.detokenize-kms", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("applying-kms", 1)
	} else {
		log.Info().Msg("already pushed kms key to gitops")
		progressPrinter.IncrementTracker("applying-kms", 1)
	}

	// Instantiate kube client for eks
	// todo create a client from config!! need to re-use AwsConfiguration or adopt session in the other direction
	progressPrinter.AddTracker("creating-eks-kube-config", "Instantiating kube client for eks", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

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

	// This flag is set if the above client config passes
	// This is used for destroy
	viper.Set("kubefirst-checks.aws-eks-cluster-created", true)

	progressPrinter.IncrementTracker("creating-eks-kube-config", 1)

	// AWS Readiness checks
	progressPrinter.AddTracker("verifying-aws-cluster-readiness", "Verifying Kubernetes cluster is ready", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// These may need to be tweaked
	// CoreDNS
	coreDNSDeployment, err := k8s.ReturnDeploymentObject(
		clientset,
		"kubernetes.io/name",
		"CoreDNS",
		"kube-system",
		120,
	)
	if err != nil {
		log.Error().Msgf("Error finding CoreDNS deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(clientset, coreDNSDeployment, 120)
	if err != nil {
		log.Error().Msgf("Error waiting for CoreDNS deployment ready state: %s", err)
		return err
	}
	progressPrinter.IncrementTracker("verifying-aws-cluster-readiness", 1)

	argocdClient, err := argocdapi.NewForConfig(restConfig)
	if err != nil {
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
	progressPrinter.AddTracker("installing-argocd", "Installing and configuring ArgoCD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	argoCDInstallPath := fmt.Sprintf("github.com:kubefirst/manifests/argocd/cloud?ref=%s", pkg.KubefirstManifestRepoRef)

	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallStarted, "")

		log.Info().Msgf("installing argocd")
		err = argocd.ApplyArgoCDKustomize(clientset, argoCDInstallPath)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallFailed, err.Error())
			return err
		}
		viper.Set("kubefirst-checks.argocd-install", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallCompleted, "")
		progressPrinter.IncrementTracker("installing-argocd", 1)
	} else {
		log.Info().Msg("argo cd already installed, continuing")
		progressPrinter.IncrementTracker("installing-argocd", 1)
	}

	// Wait for ArgoCD to be ready
	_, err = k8s.VerifyArgoCDReadiness(clientset, true, 300)
	if err != nil {
		log.Error().Msgf("error waiting for ArgoCD to become ready: %s", err)
		return err
	}
	progressPrinter.IncrementTracker("installing-argocd", 1)

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
	progressPrinter.IncrementTracker("installing-argocd", 1)

	// todo need to create argocd repo secret in the cluster
	//* create argocd kubernetes secret for connectivity to private gitops repo
	progressPrinter.AddTracker("setting-up-eks-cluster", "Setting up EKS cluster", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

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
				"name":          []byte(fmt.Sprintf("%s-gitops", cGitOwner)),
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

		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, registryURL, ecrToken)
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
		progressPrinter.IncrementTracker("setting-up-eks-cluster", 1)
	} else {
		log.Info().Msg("argo credentials already set, continuing")
		progressPrinter.IncrementTracker("setting-up-eks-cluster", 1)
	}

	var argocdPassword string
	//* argocd pods are ready, get and set credentials
	progressPrinter.AddTracker("creating-argocd-auth", "Creating ArgoCD authentication", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

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
		progressPrinter.IncrementTracker("creating-argocd-auth", 1)
	} else {
		log.Info().Msg("argo credentials already set, continuing")
		progressPrinter.IncrementTracker("creating-argocd-auth", 1)
	}

	if configs.K1Version == "development" {
		err := clipboard.WriteAll(argocdPassword)
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		err = pkg.OpenBrowser(pkg.ArgocdPortForwardURL)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
	}

	//* create registry
	progressPrinter.AddTracker("create-registry-application", "Deploying registry application to ArgoCD", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryStarted, "")

		log.Info().Msg("applying the registry application to argocd")
		registryApplicationObject := argocd.GetArgoCDApplicationObject(config.DestinationGitopsRepoGitURL, fmt.Sprintf("registry/%s", clusterNameFlag))
		_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryCompleted, "")
		progressPrinter.IncrementTracker("create-registry-application", 1)
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
		progressPrinter.IncrementTracker("create-registry-application", 1)
	}

	//* initialize and unseal vault
	progressPrinter.AddTracker("configuring-vault", "Configuring Vault", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.vault-ready")
	if !executionControl {
		log.Info().Msg("waiting for vault pods to be ready ")
		// Wait for Vault StatefulSet Pods to transition to Running
		vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
			clientset,
			"app.kubernetes.io/instance",
			"vault",
			"vault",
			600,
		)
		if err != nil {
			log.Error().Msgf("Error finding Vault StatefulSet: %s", err)
			return err
		}
		_, err = k8s.WaitForStatefulSetReady(clientset, vaultStatefulSet, 60, true)
		if err != nil {
			log.Error().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
			return err
		}

		time.Sleep(time.Second * 20) // todo remove this? might not be needed anymore
		viper.Set("kubefirst-checks.vault-ready", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("configuring-vault", 1)
	} else {
		log.Info().Msg("vault is ready, continuing")
		progressPrinter.IncrementTracker("configuring-vault", 1)
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

	executionControl = viper.GetBool("kubefirst-checks.vault-unseal")
	if !executionControl {
		log.Info().Msg("initializing vault and vault unseal")

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
		progressPrinter.IncrementTracker("configuring-vault", 1)
	} else {
		log.Info().Msg("vault unseal already done, continuing")
		progressPrinter.IncrementTracker("configuring-vault", 1)
	}

	vaultRootTokenLookup, err := k8s.ReadSecretV2(clientset, "vault", "vault-unseal-secret")
	if err != nil {
		return err
	}

	vaultRootToken = vaultRootTokenLookup["root-token"]

	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultTerraformApplyStarted, "")

		//* run vault terraform
		log.Info().Msg("configuring vault with terraform")

		tfEnvs := map[string]string{}

		usernamePasswordString := fmt.Sprintf("%s:%s", cGitUser, cGitToken)
		base64DockerAuth := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
		tfEnvs["TF_VAR_vault_addr"] = awsinternal.VaultPortForwardURL
		tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = awsinternal.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = atlantisWebhookSecret
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
		tfEnvs["TF_VAR_kbot_ssh_private_key"] = viper.GetString("kbot.private-key")
		// todo hyrdate a variable up top with these so we dont ref viper.

		if gitProviderFlag == "gitlab" {
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(viper.GetInt("flags.gitlab-owner-group-id"))
		}

		tfEntrypoint := config.GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultTerraformApplyFailed, err.Error())
			return err
		}

		log.Info().Msg("vault terraform executed successfully")
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("configuring-vault", 1)
	} else {
		log.Info().Msg("already executed vault terraform")
		progressPrinter.IncrementTracker("configuring-vault", 1)
	}

	//* create users
	progressPrinter.AddTracker("creating-users", "Creating users", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-users")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyStarted, "")

		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
		tfEnvs["TF_VAR_vault_addr"] = awsinternal.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = awsinternal.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
		tfEnvs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = cGitToken
		tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = cGitOwner

		tfEntrypoint := config.GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyStarted, err.Error())
			return err
		}
		log.Info().Msg("executed users terraform successfully")
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("creating-users", 1)
	} else {
		log.Info().Msg("already created users with terraform")
		progressPrinter.IncrementTracker("creating-users", 1)
	}

	// Wait for console Deployment Pods to transition to Running
	progressPrinter.AddTracker("deploying-kubefirst-console", "Deploying kubefirst console", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	consoleDeployment, err := k8s.ReturnDeploymentObject(
		clientset,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		1200,
	)
	if err != nil {
		log.Error().Msgf("Error finding console Deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(clientset, consoleDeployment, 120)
	if err != nil {
		log.Error().Msgf("Error waiting for console Deployment ready state: %s", err)
		return err
	}

	//* console port-forward
	progressPrinter.IncrementTracker("deploying-kubefirst-console", 1)
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
	log.Info().Msg("welcome to your new kubefirst platform powered by AWS")
	time.Sleep(time.Second * 1) // allows progress bars to finish

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	// Mark cluster install as complete
	telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricClusterInstallCompleted, "")
	viper.Set("kubefirst-checks.cluster-install-complete", true)
	viper.WriteConfig()

	// Set flags used to track status of active options
	helpers.SetClusterStatusFlags(awsinternal.CloudProvider, config.GitProvider)

	if !ciFlag {
		reports.AwsHandoffScreen(viper.GetString("components.argocd.password"), clusterNameFlag, domainNameFlag, cGitOwner, config, false)
	}

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return nil
}
