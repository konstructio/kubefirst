/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/dns"
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createCivo(cmd *cobra.Command, args []string) error {
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

	gitProtocolFlag, err := cmd.Flags().GetString("git-protocol")
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

	//Validate we got a branch if they gave us a repo
	if gitopsTemplateURLFlag != "" && gitopsTemplateBranchFlag == "" {
		log.Panic().Msgf("must supply gitops-template-branch flag when gitops-template-url is set")
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(8080, 8200, 9094)
	if err != nil {
		return fmt.Errorf("%s - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing", err.Error())
	}

	//Validate we got a branch if they gave us a repo
	if gitopsTemplateURLFlag != "" && gitopsTemplateBranchFlag == "" {
		log.Panic().Msgf("must supply gitops-template-branch flag when gitops-template-url is set")
	}

	// Validate required environment variables for dns provider
	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	// required for destroy command
	viper.Set("flags.alerts-email", alertsEmailFlag)
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.dns-provider", dnsProviderFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.git-protocol", gitProtocolFlag)
	viper.Set("flags.cloud-region", cloudRegionFlag)
	viper.WriteConfig()

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 6)
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
	config := providerConfigs.GetConfig(clusterNameFlag, domainNameFlag, gitProviderFlag, cGitOwner, gitProtocolFlag)
	config.CivoToken = os.Getenv("CIVO_TOKEN")
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
		switch gitProtocolFlag {
		case "https":
			config.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://gitlab.com/%s/gitops.git", gitlabClient.ParentGroupPath)
			config.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://gitlab.com/%s/metaphor.git", gitlabClient.ParentGroupPath)
		default:
			config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
			config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)
		}
	}

	civoConf := civo.CivoConfiguration{
		Client:  civo.NewCivo(config.CivoToken, cloudRegionFlag),
		Context: context.Background(),
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
		externalDNSProviderTokenEnvName = "CIVO_TOKEN"
		externalDNSProviderSecretKey = fmt.Sprintf("%s-token", civo.CloudProvider)
	}

	gitopsDirectoryTokens := providerConfigs.GitOpsDirectoryValues{
		AlertsEmail:               alertsEmailFlag,
		AtlantisAllowList:         fmt.Sprintf("%s/%s/*", cGitHost, cGitOwner),
		CloudProvider:             civo.CloudProvider,
		CloudRegion:               cloudRegionFlag,
		ClusterName:               clusterNameFlag,
		ClusterType:               clusterTypeFlag,
		DNSProvider:               dnsProviderFlag,
		DomainName:                domainNameFlag,
		KubeconfigPath:            config.Kubeconfig,
		KubefirstStateStoreBucket: kubefirstStateStoreBucketName,
		KubefirstTeam:             kubefirstTeam,
		KubefirstVersion:          configs.K1Version,

		ExternalDNSProviderName:         dnsProviderFlag,
		ExternalDNSProviderTokenEnvName: externalDNSProviderTokenEnvName,
		ExternalDNSProviderSecretName:   fmt.Sprintf("%s-creds", civo.CloudProvider),
		ExternalDNSProviderSecretKey:    externalDNSProviderSecretKey,

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
		GitopsRepoGitURL:     config.DestinationGitopsRepoGitURL,
		GitopsRepoHttpsURL:   config.DestinationGitopsRepoHttpsURL,
		GitopsRepoURL:        config.DestinationGitopsRepoURL,
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

		GitOpsRepoAtlantisWebhookURL: fmt.Sprintf("https://atlantis.%s/events", domainNameFlag),
		GitOpsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", cGitHost, cGitOwner),
		ClusterId:                    clusterId,
	}

	viper.Set(fmt.Sprintf("%s.atlantis.webhook.url", config.GitProvider), fmt.Sprintf("https://atlantis.%s/events", domainNameFlag))
	viper.WriteConfig()

	// Segment Client
	segmentClient := &segment.SegmentClient{
		CliVersion:        configs.K1Version,
		CloudProvider:     civo.CloudProvider,
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

		if config.CivoToken == "" {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudCredentialsCheckFailed, "CIVO_TOKEN environment variable was not set")
			return fmt.Errorf("your CIVO_TOKEN is not set - please set and re-run your last command")
		}
		log.Info().Msg("CIVO_TOKEN set - continuing")
		viper.Set("kubefirst-checks.cloud-credentials", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudCredentialsCheckCompleted, "")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already completed cloud credentials check - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	executionControl = viper.GetBool("kubefirst-checks.state-store-creds")
	if !executionControl {
		creds, err := civoConf.GetAccessCredentials(kubefirstStateStoreBucketName, cloudRegionFlag)
		if err != nil {
			log.Info().Msg(err.Error())
		}

		viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyID)
		viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKeyID)
		viper.Set("kubefirst.state-store-creds.name", creds.Name)
		viper.Set("kubefirst.state-store-creds.id", creds.ID)
		viper.Set("kubefirst-checks.state-store-creds", true)
		viper.WriteConfig()
		log.Info().Msg("civo object storage credentials created and set")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already created civo object storage credentials - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	skipDomainCheck := viper.GetBool("kubefirst-checks.domain-liveness")
	if !skipDomainCheck {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessStarted, "")

		switch dnsProviderFlag {
		case "civo":
			// verify dns
			err := dns.VerifyProviderDNS(civo.CloudProvider, cloudRegionFlag, domainNameFlag, nil)
			if err != nil {
				return err
			}

			// domain id
			domainId, err := civoConf.GetDNSInfo(domainNameFlag, cloudRegionFlag)
			if err != nil {
				log.Info().Msg(err.Error())
			}

			// viper values set in above function
			log.Info().Msgf("domainId: %s", domainId)
			domainLiveness := civoConf.TestDomainLiveness(domainNameFlag, domainId, cloudRegionFlag)
			if !domainLiveness {
				telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessFailed, "domain liveness test failed")
				msg := "failed to check the liveness of the Domain. A valid public Domain on the same CIVO " +
					"account as the one where Kubefirst will be installed is required for this operation to " +
					"complete.\nTroubleshoot Steps:\n\n - Make sure you are using the correct CIVO account and " +
					"region.\n - Verify that you have the necessary permissions to access the domain.\n - Check " +
					"that the domain is correctly configured and is a public domain\n - Check if the " +
					"domain exists and has the correct name and domain.\n - If you don't have a Domain," +
					"please follow these instructions to create one: " +
					"https://www.civo.com/learn/configure-dns \n\n" +
					"if you are still facing issues please reach out to support team for further assistance"

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

		accessKeyId := viper.GetString("kubefirst.state-store-creds.access-key-id")
		log.Info().Msgf("access key id %s", accessKeyId)

		bucket, err := civoConf.CreateStorageBucket(accessKeyId, kubefirstStateStoreBucketName, cloudRegionFlag)
		if err != nil {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
			log.Info().Msg(err.Error())
			return err
		}

		viper.Set("kubefirst.state-store.id", bucket.ID)
		viper.Set("kubefirst.state-store.name", bucket.Name)
		viper.Set("kubefirst-checks.state-store-create", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateCompleted, "")
		log.Info().Msg("civo state store bucket created")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already created civo state store bucket - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	// Check quotas
	quotaMessage, quotaFailures, quotaWarnings, err := returnCivoQuotaEvaluation(cloudRegionFlag)
	if err != nil {
		return err
	}
	switch {
	case quotaFailures > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
		return fmt.Errorf("at least one of your Civo quotas is close to its limit. Please check the error message above for additional details")
	case quotaWarnings > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
	}
	//* CIVO END

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

		err := civo.DownloadTools(
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

	progressPrinter.AddTracker("cloning-and-formatting-git-repositories", "Cloning and formatting git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		log.Info().Msg("generating your new gitops repository")

		// These need to be set for reference elsewhere
		viper.Set(fmt.Sprintf("%s.repos.gitops.git-url", config.GitProvider), config.DestinationGitopsRepoURL)
		viper.WriteConfig()

		// Determine if anything exists at domain apex
		apexContentExists := civo.GetDomainApexContent(domainNameFlag)

		err = providerConfigs.PrepareGitRepositories(
			civo.CloudProvider,
			config.GitProvider,
			clusterNameFlag,
			clusterTypeFlag,
			config.DestinationGitopsRepoHttpsURL,
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			config.DestinationMetaphorRepoHttpsURL,
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
	log.Info().Msgf("referencing gitops repository: %s", config.DestinationGitopsRepoHttpsURL)
	log.Info().Msgf("referencing metaphor repository: %s", config.DestinationMetaphorRepoHttpsURL)
	switch config.GitProvider {
	case "github":
		// //* create teams and repositories in github
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
		if !executionControl {
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitHub resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = civo.GetGithubTerraformEnvs(config, tfEnvs)
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
			tfEnvs = civo.GetGitlabTerraformEnvs(config, tfEnvs, cGitlabOwnerGroupID)
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

	log.Info().Msgf("referencing gitops repository: %s", config.DestinationGitopsRepoHttpsURL)
	log.Info().Msgf("referencing metaphor repository: %s", config.DestinationMetaphorRepoHttpsURL)

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
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", config.DestinationGitopsRepoHttpsURL, err)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			if !strings.Contains(msg, "already up-to-date") {
				log.Panic().Msg(msg)
			}
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       httpAuth,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized metaphor repository to remote %s: %s", config.DestinationMetaphorRepoHttpsURL, err)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricGitopsRepoPushFailed, msg)
			if !strings.Contains(msg, "already up-to-date") {
				log.Panic().Msg(msg)
			}
		}
		log.Info().Msgf("successfully pushed gitops and metaphor repositories to https://%s/%s", cGitHost, cGitOwner)

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

	//* create civo cloud resources
	progressPrinter.AddTracker("applying-civo-terraform", "Applying Civo Terraform", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyStarted, "")

		log.Info().Msg("Creating civo cloud resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(config, tfEnvs)
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			msg := fmt.Sprintf("error creating civo resources with terraform %s : %s", tfEntrypoint, err)
			viper.Set("kubefirst-checks.terraform-apply-civo-failed", true)
			viper.WriteConfig()
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyFailed, msg)
			return fmt.Errorf(msg)
		}

		log.Info().Msg("Created civo cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-civo", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("applying-civo-terraform", 1)
	} else {
		log.Info().Msg("already created GitHub Terraform resources")
		progressPrinter.IncrementTracker("applying-civo-terraform", 1)
	}

	//* civo needs extra time to be ready
	progressPrinter.AddTracker("wait-for-civo", "Wait for Civo Kubernetes", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.k8s-secrets-created") {
		time.Sleep(time.Second * 60)
	} else {
		time.Sleep(time.Second * 5)
	}
	progressPrinter.IncrementTracker("wait-for-civo", 1)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	// Civo Readiness checks
	progressPrinter.AddTracker("verifying-civo-cluster-readiness", "Verifying Kubernetes cluster is ready", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// CoreDNS
	coreDNSDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"kubernetes.io/name",
		"CoreDNS",
		"kube-system",
		240,
	)
	if err != nil {
		log.Error().Msgf("Error finding CoreDNS deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, coreDNSDeployment, 240)
	if err != nil {
		log.Error().Msgf("Error waiting for CoreDNS deployment ready state: %s", err)
		return err
	}
	progressPrinter.IncrementTracker("verifying-civo-cluster-readiness", 1)

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	progressPrinter.AddTracker("bootstrapping-kubernetes-resources", "Bootstrapping Kubernetes resources", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := civo.BootstrapCivoMgmtCluster(
			config.CivoToken,
			config.Kubeconfig,
			config.GitProvider,
			cGitUser,
			os.Getenv("CF_API_TOKEN"),
			config.DestinationGitopsRepoURL,
			config.GitProtocol,
		)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	} else {
		log.Info().Msg("already added secrets to civo cluster")
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
		registryApplicationObject := argocd.GetArgoCDApplicationObject(config.DestinationGitopsRepoURL, fmt.Sprintf("registry/%s", clusterNameFlag))
		_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCreateRegistryCompleted, "")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	}

	// Wait for Vault StatefulSet Pods to transition to Running
	progressPrinter.AddTracker("configuring-vault", "Configuring Vault", 4)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		240,
	)
	if err != nil {
		log.Error().Msgf("Error finding Vault StatefulSet: %s", err)
		return err
	}
	_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 240, true)
	if err != nil {
		log.Error().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
		return err
	}
	progressPrinter.IncrementTracker("configuring-vault", 1)

	// Init and unseal vault
	// We need to wait before we try to run any of these commands or there may be
	// unexpected timeouts
	time.Sleep(time.Second * 10)
	progressPrinter.IncrementTracker("configuring-vault", 1)

	executionControl = viper.GetBool("kubefirst-checks.vault-initialized")
	if !executionControl {
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultInitializationStarted, "")

		// Initialize and unseal Vault
		vaultHandlerPath := "github.com:kubefirst/manifests.git/vault-handler/replicas-3"

		// Build and apply manifests
		yamlData, err := kcfg.KustomizeBuild(vaultHandlerPath)
		if err != nil {
			return err
		}
		output, err := kcfg.SplitYAMLFile(yamlData)
		if err != nil {
			return err
		}
		err = kcfg.ApplyObjects("", output)
		if err != nil {
			return err
		}

		// Wait for the Job to finish
		job, err := k8s.ReturnJobObject(kcfg.Clientset, "vault", "vault-handler")
		if err != nil {
			return err
		}
		_, err = k8s.WaitForJobComplete(kcfg.Clientset, job, 240)
		if err != nil {
			msg := fmt.Sprintf("could not run vault unseal job: %s", err)
			telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultInitializationFailed, msg)
			log.Fatal().Msg(msg)
		}

		viper.Set("kubefirst-checks.vault-initialized", true)
		viper.WriteConfig()
		telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricVaultInitializationCompleted, "")
		progressPrinter.IncrementTracker("configuring-vault", 1)
	} else {
		log.Info().Msg("vault is already initialized - skipping")
		progressPrinter.IncrementTracker("configuring-vault", 1)
	}

	//* configure vault with terraform
	//* vault port-forward
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

		tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth
		tfEnvs = civo.GetVaultTerraformEnvs(kcfg.Clientset, config, tfEnvs)
		tfEnvs = civo.GetCivoTerraformEnvs(config, tfEnvs)
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
		tfEnvs = civo.GetCivoTerraformEnvs(config, tfEnvs)
		tfEnvs = civo.GetUsersTerraformEnvs(kcfg.Clientset, config, tfEnvs)
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
	log.Info().Msg("welcome to your new kubefirst platform powered by Civo cloud")
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
	helpers.SetClusterStatusFlags(civo.CloudProvider, config.GitProvider)

	if !ciFlag {
		reports.CivoHandoffScreen(viper.GetString("components.argocd.password"), clusterNameFlag, domainNameFlag, cGitOwner, config, false)
	}

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return nil
}
