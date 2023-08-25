/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/rs/zerolog/log"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kubefirst/kubefirst/internal/gitShim"
	"github.com/kubefirst/kubefirst/internal/telemetryShim"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	"github.com/kubefirst/runtime/pkg/gcp"
	"github.com/kubefirst/runtime/pkg/github"
	gitlab "github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/handlers"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/progressPrinter"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/reports"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/services"
	internalssh "github.com/kubefirst/runtime/pkg/ssh"
	"github.com/kubefirst/runtime/pkg/ssl"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/kubefirst/runtime/pkg/vault"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thanhpk/randstr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func createGCP(cmd *cobra.Command, args []string) error {
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

	dnsProviderFlag, err := cmd.Flags().GetString("dns-provider")
	if err != nil {
		return err
	}

	domainNameFlag, err := cmd.Flags().GetString("domain-name")
	if err != nil {
		return err
	}

	forceDestroy, err := cmd.Flags().GetBool("force_destroy")
	if err != nil {
		return err
	}

	gcpProjectFlag, err := cmd.Flags().GetString("gcp-project")
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

	// required for destroy command
	viper.Set("flags.alerts-email", alertsEmailFlag)
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.dns-provider", dnsProviderFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.cloud-region", cloudRegionFlag)
	viper.Set("flags.gcp-project", gcpProjectFlag)
	viper.WriteConfig()

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 5)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// Switch based on git provider, set params
	var cGitHost, cGitOwner, cGitToken, cGitUser, containerRegistryHost string
	var cGitlabOwnerGroupID int
	switch gitProviderFlag {
	case "github":
		if githubOrgFlag == "" {
			return fmt.Errorf("please provide a github organization using the --github-org flag")
		}
		if os.Getenv("GITHUB_TOKEN") == "" {
			return fmt.Errorf("your GITHUB_TOKEN is not set. Please set and try again")
		}

		cGitHost = providerConfigs.GithubHost
		cGitOwner = githubOrgFlag
		cGitToken = os.Getenv("GITHUB_TOKEN")
		containerRegistryHost = "ghcr.io"

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

		cGitHost = providerConfigs.GitlabHost
		cGitOwner = gitlabClient.ParentGroupPath
		cGitlabOwnerGroupID = gitlabClient.ParentGroupID
		log.Info().Msgf("set gitlab owner to %s", cGitOwner)

		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		cGitUser = user.Username

		containerRegistryHost = "registry.gitlab.com"
		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
		viper.Set("flags.gitlab-owner-group-id", cGitlabOwnerGroupID)
		viper.WriteConfig()
	default:
		log.Error().Msgf("invalid git provider option")
	}

	// Instantiate config
	config := providerConfigs.GetConfig(
		clusterNameFlag,
		domainNameFlag,
		gitProviderFlag,
		cGitOwner,
		gitProtocolFlag,
		os.Getenv("CF_API_TOKEN"),
	)
	// This is the environment variable required to create and is set to the path of the service account json file
	// This gets read for terraform applies and is applied as a variable containing the contents of the file
	// This is otherwise leveraged by the runtime to provide application default credentials to the GCP go SDK/API
	config.GCPAuth = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	switch gitProviderFlag {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
		// gitlab may have subgroups, so the destination gitops/metaphor repo git urls may be different
		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
		if err != nil {
			return err
		}
		// Format git url based on full path to group
		config.DestinationGitopsRepoURL = fmt.Sprintf("https://gitlab.com/%s/gitops.git", gitlabClient.ParentGroupPath)
		config.DestinationMetaphorRepoURL = fmt.Sprintf("https://gitlab.com/%s/metaphor.git", gitlabClient.ParentGroupPath)
		config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
		config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)
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

	// Detokenize
	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	var externalDNSProviderTokenEnvName, externalDNSProviderSecretKey string
	if dnsProviderFlag == "cloudflare" {
		externalDNSProviderTokenEnvName = "CF_API_TOKEN"
		externalDNSProviderSecretKey = "cf-api-token"
	} else {
		externalDNSProviderTokenEnvName = "GOOGLE_AUTH"
		externalDNSProviderSecretKey = fmt.Sprintf("%s-auth", dnsProviderFlag)
	}

	// Swap tokens for git protocol
	var gitopsRepoURL string
	switch config.GitProtocol {
	case "https":
		gitopsRepoURL = config.DestinationGitopsRepoURL
	default:
		gitopsRepoURL = config.DestinationGitopsRepoGitURL
	}

	gitopsDirectoryTokens := providerConfigs.GitopsDirectoryValues{
		AlertsEmail:               alertsEmailFlag,
		AtlantisAllowList:         fmt.Sprintf("%s/%s/*", cGitHost, cGitOwner),
		CloudProvider:             gcp.CloudProvider,
		CloudRegion:               cloudRegionFlag,
		ClusterName:               clusterNameFlag,
		ClusterType:               clusterTypeFlag,
		DNSProvider:               dnsProviderFlag,
		DomainName:                domainNameFlag,
		KubeconfigPath:            config.Kubeconfig,
		KubefirstArtifactsBucket:  kubefirstArtifactsBucketName,
		KubefirstStateStoreBucket: kubefirstStateStoreBucketName,
		KubefirstTeam:             kubefirstTeam,
		KubefirstVersion:          configs.K1Version,

		GCPAuth:          config.GCPAuth,
		GCPProject:       gcpProjectFlag,
		GoogleUniqueness: strings.ToLower(randstr.String(5)),
		ForceDestroy:     strconv.FormatBool(forceDestroy),

		ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", domainNameFlag),
		ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", domainNameFlag),
		ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", domainNameFlag),
		ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", domainNameFlag),
		AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", domainNameFlag),
		AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", domainNameFlag),
		ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", domainNameFlag),
		VaultIngressURL:                fmt.Sprintf("https://vault.%s", domainNameFlag),
		VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", domainNameFlag),
		VaultDataBucketName:            fmt.Sprintf("%s-vault-data-%s", gcpProjectFlag, clusterNameFlag),
		VouchIngressURL:                fmt.Sprintf("https://vouch.%s", domainNameFlag),

		ExternalDNSProviderName:         dnsProviderFlag,
		ExternalDNSProviderTokenEnvName: externalDNSProviderTokenEnvName,
		ExternalDNSProviderSecretName:   fmt.Sprintf("%s-auth", dnsProviderFlag),
		ExternalDNSProviderSecretKey:    externalDNSProviderSecretKey,

		GitDescription:       fmt.Sprintf("%s hosted git", config.GitProvider),
		GitNamespace:         "N/A",
		GitProvider:          config.GitProvider,
		GitProtocol:          config.GitProtocol,
		GitopsRepoURL:        gitopsRepoURL,
		GitRunner:            fmt.Sprintf("%s Runner", config.GitProvider),
		GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", config.GitProvider),
		GitRunnerNS:          fmt.Sprintf("%s-runner", config.GitProvider),
		GitURL:               gitopsTemplateURLFlag,

		GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", cGitOwner),
		GitHubOwner: cGitOwner,
		GitHubUser:  cGitUser,

		GitlabHost:         providerConfigs.GitlabHost,
		GitlabOwner:        cGitOwner,
		GitlabOwnerGroupID: cGitlabOwnerGroupID,
		GitlabUser:         cGitUser,

		GitopsRepoAtlantisWebhookURL: fmt.Sprintf("https://atlantis.%s/events", domainNameFlag),
		GitopsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", cGitHost, cGitOwner),
		ClusterId:                    clusterId,

		ContainerRegistryURL: fmt.Sprintf("%s/%s/metaphor", containerRegistryHost, cGitOwner),
	}

	viper.Set(fmt.Sprintf("%s.atlantis.webhook.url", config.GitProvider), fmt.Sprintf("https://atlantis.%s/events", domainNameFlag))
	viper.WriteConfig()

	// Segment Client
	segmentClient := &segment.SegmentClient{
		CliVersion:        configs.K1Version,
		CloudProvider:     gcp.CloudProvider,
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
	if useTelemetryFlag {
		gitopsDirectoryTokens.UseTelemetry = "true"

		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricInitStarted, "")
	} else {
		gitopsDirectoryTokens.UseTelemetry = "false"
	}

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
			if gitopsTemplateBranchFlag != "" {
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

	executionControl := viper.GetBool("kubefirst-checks.cloud-credentials")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudCredentialsCheckStarted, "")

		if config.GCPAuth == "" {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudCredentialsCheckFailed, "GOOGLE_APPLICATION_CREDENTIALS environment variable was not set")
			return fmt.Errorf("your GOOGLE_APPLICATION_CREDENTIALS variable is unset - please set it before continuing")
		}
		viper.Set("kubefirst-checks.cloud-credentials", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudCredentialsCheckCompleted, "")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already completed cloud credentials check - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	skipDomainCheck := viper.GetBool("kubefirst-checks.domain-liveness")
	if !skipDomainCheck {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessStarted, "")

		switch dnsProviderFlag {
		case "gcp":
			gcpConf := gcp.GCPConfiguration{
				Context: context.Background(),
				Project: gcpProjectFlag,
				Region:  cloudRegionFlag,
			}

			// verify dns
			// TODO: update to work with gcp?
			//err := dns.VerifyProviderDNS(gcp.CloudProvider, cloudRegionFlag, domainNameFlag)
			//if err != nil {
			//	return err
			//}

			// viper values set in above function
			domainLiveness := gcpConf.TestHostedZoneLiveness(domainNameFlag)
			if !domainLiveness {
				telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessFailed, "domain liveness test failed")
				msg := "failed to check the liveness of the Domain. A valid public Domain on the same GCP " +
					"account as the one where Kubefirst will be installed is required for this operation to " +
					"complete.\nTroubleshoot Steps:\n\n - Make sure you are using the correct GCP project and " +
					"region.\n - Verify that you have the necessary permissions to access the domain.\n - Check " +
					"that the domain is correctly configured and is a public domain\n - Check if the " +
					"domain exists and has the correct name and domain.\n - If you don't have a Domain," +
					"please follow these instructions to create one: " +
					"https://cloud.google.com/dns/docs/tutorials/create-domain-tutorial \n\n" +
					"if you are still facing issues please reach out to the support team for further assistance"

				return fmt.Errorf(msg)
			}
			viper.Set("kubefirst-checks.domain-liveness", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessCompleted, "")
			progressPrinter.IncrementTracker("preflight-checks", 1)
		case "cloudflare":
			// Implement a Cloudflare check at some point
			log.Info().Msg("domain check already complete - continuing")
			progressPrinter.IncrementTracker("preflight-checks", 1)
		}
	} else {
		log.Info().Msg("domain check already complete - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	executionControl = viper.GetBool("kubefirst-checks.state-store-create")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateStarted, "")

		gcpConf := gcp.GCPConfiguration{
			Context: context.Background(),
			Project: gcpProjectFlag,
			Region:  cloudRegionFlag,
		}

		// state store
		stobjst, err := gcpConf.CreateBucket(kubefirstStateStoreBucketName)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
			log.Info().Msg(err.Error())
			return err
		}

		// artifacts store
		aobjst, err := gcpConf.CreateBucket(kubefirstArtifactsBucketName)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
			log.Info().Msg(err.Error())
			return err
		}

		viper.Set("kubefirst.state-store-bucket", stobjst.Name)
		viper.Set("kubefirst.artifacts-bucket", aobjst.Name)
		viper.Set("kubefirst-checks.state-store-create", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateCompleted, "")

		log.Info().Msg("gcp state store bucket created")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already created gcp state store bucket - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	// Objects to check for
	newRepositoryNames := []string{"gitops", "metaphor"}
	newTeamNames := []string{"admins", "developers"}

	executionControl = viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider))
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
	progressPrinter.IncrementTracker("preflight-checks", 1)
	progressPrinter.IncrementTracker("preflight-checks", 1)
	telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricInitCompleted, "")
	telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricClusterInstallStarted, "")

	//removed because we no longer default to ssh for kubefirst cli calls since we require the token anyways
	// publicKeys, err := ssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	// if err != nil {
	// 	log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	// }

	//* generate http credentials for git auth over https
	httpAuth := &githttps.BasicAuth{
		Username: cGitUser,
		Password: cGitToken,
	}

	//* download dependencies to `$HOME/.k1/tools`
	progressPrinter.AddTracker("downloading-tools", "Downloading tools", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := gcp.DownloadTools(
			config.KubectlClient,
			providerConfigs.KubectlClientVersion,
			providerConfigs.LocalhostOS,
			providerConfigs.LocalhostArch,
			providerConfigs.TerraformClientVersion,
			config.ToolsDir,
		)
		if err != nil {
			return err
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("downloading-tools", 1)
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
		progressPrinter.IncrementTracker("downloading-tools", 1)
	}

	// todo should metaphor tokens be global?
	metaphorDirectoryTokens := providerConfigs.MetaphorTokenValues{
		ClusterName:                   clusterNameFlag,
		CloudRegion:                   cloudRegionFlag,
		ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", containerRegistryHost, cGitOwner),
		DomainName:                    domainNameFlag,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", domainNameFlag),
		MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", domainNameFlag),
		MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", domainNameFlag),
	}

	config.GitopsDirectoryValues = &gitopsDirectoryTokens
	config.MetaphorDirectoryValues = &metaphorDirectoryTokens
	//* git clone and detokenize the gitops repository

	progressPrinter.AddTracker("cloning-and-formatting-git-repositories", "Cloning and formatting git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		log.Info().Msg("generating your new gitops repository")

		// These need to be set for reference elsewhere
		viper.Set(fmt.Sprintf("%s.repos.gitops.git-url", config.GitProvider), config.DestinationGitopsRepoURL)
		viper.WriteConfig()

		// Determine if anything exists at domain apex
		apexContentExists := gcp.GetDomainApexContent(domainNameFlag)

		err = providerConfigs.PrepareGitRepositories(
			gcp.CloudProvider,
			config.GitProvider,
			clusterNameFlag,
			clusterTypeFlag,
			config.DestinationGitopsRepoURL, //default to https for git interactions when creating remotes
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			config.DestinationMetaphorRepoURL, //default to https for git interactions when creating remotes
			config.K1Dir,
			&gitopsDirectoryTokens,
			config.MetaphorDir,
			&metaphorDirectoryTokens,
			apexContentExists,
			gitProtocolFlag,
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

	//* handle git terraform apply
	progressPrinter.AddTracker("applying-git-terraform", fmt.Sprintf("Applying %s Terraform", config.GitProvider), 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	switch config.GitProvider {
	case "github":
		// //* create teams and repositories in github
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
		if !executionControl {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitHub resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			a, _ := os.ReadFile(config.GCPAuth)
			tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
			tfEnvs = gcp.GetGithubTerraformEnvs(config, tfEnvs)
			// Erase public key to prevent it from being created if the git protocol argument is set to htps
			switch config.GitProtocol {
			case "https":
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
			}
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
			log.Info().Msg("already created GitHub Terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	case "gitlab":
		// //* create teams and repositories in gitlab
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-gitlab")
		if !executionControl {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitLab resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			a, _ := os.ReadFile(config.GCPAuth)
			tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
			tfEnvs = gcp.GetGitlabTerraformEnvs(config, tfEnvs, cGitlabOwnerGroupID)
			// Erase public key to prevent it from being created if the git protocol argument is set to htps
			switch config.GitProtocol {
			case "https":
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
			}
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
			log.Info().Msg("already created GitLab Terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	}

	//* push detokenized gitops-template repository content to new remote
	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %s", config.DestinationGitopsRepoURL)
	log.Info().Msgf("referencing metaphor repository: %s", config.DestinationMetaphorRepoURL)

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
			Auth:       httpAuth,
		})
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", config.DestinationGitopsRepoURL, err)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			log.Panic().Msg(msg)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       httpAuth,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized metaphor repository to remote %s: %s", config.DestinationMetaphorRepoURL, err)
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

	//* create gcp cloud resources
	progressPrinter.AddTracker("applying-gcp-terraform", "Applying GCP Terraform", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.terraform-apply-gcp") {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

		log.Info().Msg("Creating GCP cloud resources with terraform")

		// Enable APIs first
		tfEnvs := map[string]string{}
		a, _ := os.ReadFile(config.GCPAuth)
		tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
		tfEnvs["TF_VAR_project"] = gcpProjectFlag
		tfEntrypoint := config.GitopsDir + "/terraform/gcp/services"
		err = terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			msg := fmt.Sprintf("error creating gcp api enable resources with terraform %s : %s", tfEntrypoint, err)
			viper.Set("kubefirst-checks.terraform-apply-gcp-failed", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
			return fmt.Errorf(msg)
		}

		// Apply everything else
		tfEnvs = map[string]string{}
		tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
		tfEntrypoint = config.GitopsDir + "/terraform/gcp"
		err = terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			msg := fmt.Sprintf("error creating gcp resources with terraform %s : %s", tfEntrypoint, err)
			viper.Set("kubefirst-checks.terraform-apply-gcp-failed", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
			return fmt.Errorf(msg)
		}

		log.Info().Msg("Created GCP cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-gcp", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("applying-gcp-terraform", 1)
	} else {
		log.Info().Msg("already created GitHub Terraform resources")
		progressPrinter.IncrementTracker("applying-gcp-terraform", 1)
	}

	// This flag is set if the above client config passes
	// This is used for destroy
	viper.Set("kubefirst-checks.gcp-kubernetes-cluster-created", true)
	viper.WriteConfig()

	//* GCP needs extra time to be ready
	progressPrinter.AddTracker("wait-for-gcp", "Wait for GCP Kubernetes", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.k8s-secrets-created") {
		time.Sleep(time.Second * 30)
	} else {
		time.Sleep(time.Second * 5)
	}
	progressPrinter.IncrementTracker("wait-for-gcp", 1)

	gcpConf := gcp.GCPConfiguration{
		Context: context.Background(),
		Project: gcpProjectFlag,
		Region:  cloudRegionFlag,
	}
	kcfg, err := gcpConf.GetContainerClusterAuth(clusterNameFlag)
	if err != nil {
		return fmt.Errorf("could not build kubernetes config for gcp cluster %s: %s", clusterNameFlag, err)
	}

	// kubernetes.BootstrapSecrets
	progressPrinter.AddTracker("bootstrapping-kubernetes-resources", "Bootstrapping Kubernetes resources", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := gcp.BootstrapGCPMgmtCluster(
			kcfg.Clientset,
			config.GitProvider,
			cGitUser,
			gitopsRepoURL,
			config.GitProtocol,
			os.Getenv("CF_API_TOKEN"),
			config.GCPAuth,
			dnsProviderFlag,
			gitopsDirectoryTokens.CloudProvider,
		)

		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	} else {
		log.Info().Msg("already added secrets to GCP cluster")
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	}

	//* check for ssl restore
	log.Info().Msg("checking for tls secrets to restore")
	secretsFilesToRestore, err := ioutil.ReadDir(config.SSLBackupDir + "/secrets")
	if err != nil {
		log.Info().Msgf("%s", err)
	}
	if len(secretsFilesToRestore) != 0 {
		// todo would like these but requires CRD's and is not currently supported
		// add crds ( use execShellReturnErrors? )
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
		// add certificates, and clusterissuers
		log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
		ssl.Restore(config.SSLBackupDir, domainNameFlag, config.Kubeconfig)
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	} else {
		log.Info().Msg("no files found in secrets directory, continuing")
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	}

	// Container registry authentication creation
	containerRegistryAuth := gitShim.ContainerRegistryAuth{
		GitProvider:           gitProviderFlag,
		GitUser:               cGitUser,
		GitToken:              cGitToken,
		GitlabGroupFlag:       gitlabGroupFlag,
		GithubOwner:           cGitOwner,
		ContainerRegistryHost: containerRegistryHost,
		Clientset:             kcfg.Clientset,
	}
	containerRegistryAuthToken, err := gitShim.CreateContainerRegistrySecret(&containerRegistryAuth)
	if err != nil {
		return err
	}

	progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	progressPrinter.AddTracker("installing-argo-cd", "Installing and configuring Argo CD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	argoCDInstallPath := fmt.Sprintf("github.com:kubefirst/manifests/argocd/cloud?ref=%s", pkg.KubefirstManifestRepoRef)

	//* install argocd
	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallStarted, "")

		log.Info().Msgf("installing argocd")
		err = argocd.ApplyArgoCDKustomize(kcfg.Clientset, argoCDInstallPath)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallFailed, err.Error())
			return err
		}
		viper.Set("kubefirst-checks.argocd-install", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricArgoCDInstallCompleted, "")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argo cd already installed, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	}

	// Wait for ArgoCD to be ready
	_, err = k8s.VerifyArgoCDReadiness(kcfg.Clientset, true, 300)
	if err != nil {
		log.Error().Msgf("error waiting for ArgoCD to become ready: %s", err)
		return err
	}

	//* ArgoCD port-forward
	argoCDStopChannel := make(chan struct{}, 1)
	defer func() {
		close(argoCDStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		"argocd-server", // todo fix this, it should `argocd
		"argocd",
		8080,
		8080,
		argoCDStopChannel,
	)
	log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

	//* argocd pods are ready, get and set credentials
	var argocdPassword string
	executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	if !executionControl {
		log.Info().Msg("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = kcfg.Clientset.CoreV1().Secrets("argocd")

		argocdPassword = k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
		if argocdPassword == "" {
			log.Info().Msg("argocd password not found in secret")
			return err
		}

		viper.Set("components.argocd.password", argocdPassword)
		viper.Set("components.argocd.username", "admin")
		viper.WriteConfig()
		log.Info().Msg("argocd username and password credentials set successfully")

		log.Info().Msg("Getting an argocd auth token")
		// todo return in here and pass argocdAuthToken as a parameter
		token, err := argocd.GetArgoCDToken("admin", argocdPassword)
		if err != nil {
			return err
		}

		log.Info().Msg("argocd admin auth token set")

		viper.Set("components.argocd.auth-token", token)
		viper.Set("kubefirst-checks.argocd-credentials-set", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argo credentials already set, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
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

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryStarted, "")
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return err
		}

		log.Info().Msg("applying the registry application to argocd")
		registryApplicationObject := argocd.GetArgoCDApplicationObject(gitopsRepoURL, fmt.Sprintf("registry/%s", clusterNameFlag))

		_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryCompleted, "")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	}

	//* initialize and unseal vault
	progressPrinter.AddTracker("configuring-vault", "Configuring Vault", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.vault-ready")
	if !executionControl {
		log.Info().Msg("waiting for vault pods to be ready ")
		// Wait for Vault StatefulSet Pods to transition to Running
		vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
			kcfg.Clientset,
			"app.kubernetes.io/instance",
			"vault",
			"vault",
			600,
		)
		if err != nil {
			log.Error().Msgf("Error finding Vault StatefulSet: %s", err)
			return err
		}
		_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 60, true)
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
	vaultClient := &vault.Conf
	var vaultRootToken string

	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
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

		err = k8s.CreateSecretV2(kcfg.Clientset, &secret)
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

	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultTerraformApplyStarted, "")

		tfEnvs := map[string]string{}
		var usernamePasswordString, base64DockerAuth string

		//* run vault terraform
		log.Info().Msg("configuring vault with terraform")
		if config.GitProvider == "gitlab" {
			usernamePasswordString = fmt.Sprintf("%s:%s", "container-registry-auth", containerRegistryAuthToken)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

			tfEnvs["TF_VAR_container_registry_auth"] = containerRegistryAuthToken
		} else {
			usernamePasswordString = fmt.Sprintf("%s:%s", cGitUser, cGitToken)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		}

		a, _ := os.ReadFile(config.GCPAuth)
		tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
		tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth
		tfEnvs = gcp.GetVaultTerraformEnvs(kcfg.Clientset, config, tfEnvs)
		tfEnvs = gcp.GetGCPTerraformEnvs(config, tfEnvs)
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
		a, _ := os.ReadFile(config.GCPAuth)
		tfEnvs["GOOGLE_CLOUD_KEYFILE_JSON"] = string(a)
		tfEnvs = gcp.GetGCPTerraformEnvs(config, tfEnvs)
		tfEnvs = gcp.GetUsersTerraformEnvs(kcfg.Clientset, config, tfEnvs)
		tfEntrypoint := config.GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricUsersTerraformApplyStarted, err.Error())
			return err
		}
		log.Info().Msg("executed users terraform successfully")
		// progressPrinter.IncrementTracker("step-users", 1)
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
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		1200,
	)
	if err != nil {
		log.Error().Msgf("Error finding console Deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, consoleDeployment, 240)
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
		kcfg.Clientset,
		kcfg.RestConfig,
		"kubefirst-console",
		"kubefirst",
		8080,
		9094,
		consoleStopChannel,
	)

	log.Info().Msg("kubefirst installation complete")
	log.Info().Msg("welcome to your new kubefirst platform powered by GCP cloud")
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
	helpers.SetClusterStatusFlags(gcp.CloudProvider, config.GitProvider)

	if !ciFlag {
		reports.GCPHandoffScreen(viper.GetString("components.argocd.password"), clusterNameFlag, domainNameFlag, cGitOwner, config, false)
	}

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return nil
}
