package civo

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/term"
	v1 "k8s.io/api/core/v1"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/github"
	gitlab "github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/segment"
	"github.com/kubefirst/kubefirst/internal/services"
	internalssh "github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createCivo(cmd *cobra.Command, args []string) error {

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 6)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	alertsEmailFlag, err := cmd.Flags().GetString("alerts-email")
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

	dryRunFlag, err := cmd.Flags().GetBool("dry-run")
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

	kbotPasswordFlag, err := cmd.Flags().GetString("kbot-password")
	if err != nil {
		return err
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return err
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(8080, 8200, 9094)
	if err != nil {
		log.Fatal().Msgf("%s - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing", err.Error())
		return err
	}

	// required for destroy command
	viper.Set("flags.alerts-email", alertsEmailFlag)
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.dry-run", dryRunFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.cloud-region", cloudRegionFlag)
	viper.WriteConfig()

	segmentClient := &segment.Client

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	// Switch based on git provider, set params
	var cGitHost, cGitOwner, cGitToken, cGitUser, containerRegistryHost string
	var cGitlabOwnerGroupID int
	switch gitProviderFlag {
	case "github":
		if os.Getenv("GITHUB_TOKEN") == "" {
			return errors.New("your GITHUB_TOKEN is not set. Please set and try again")
		}

		cGitHost = civo.GithubHost
		cGitOwner = githubOrgFlag
		cGitToken = os.Getenv("GITHUB_TOKEN")
		containerRegistryHost = "ghcr.io"

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
		if os.Getenv("GITLAB_TOKEN") == "" {
			return errors.New("your GITLAB_TOKEN is not set. please set and try again")
		}

		cGitToken = os.Getenv("GITLAB_TOKEN")
		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
		if err != nil {
			return err
		}

		cGitHost = civo.GitlabHost
		cGitOwner = gitlabClient.ParentGroupPath
		log.Info().Msgf("set gitlab owner to %s", cGitOwner)

		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		cGitUser = user.Username

		containerRegistryHost = "registry.gitlab.com"
		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
		viper.WriteConfig()
	default:
		log.Error().Msgf("invalid git provider option")
	}

	// Instantiate config
	config := civo.GetConfig(clusterNameFlag, domainNameFlag, gitProviderFlag, cGitOwner)

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

	gitopsDirectoryTokens := civo.GitOpsDirectoryValues{
		AlertsEmail:               alertsEmailFlag,
		AtlantisAllowList:         fmt.Sprintf("%s/%s/*", cGitHost, cGitOwner),
		CloudProvider:             civo.CloudProvider,
		CloudRegion:               cloudRegionFlag,
		ClusterName:               clusterNameFlag,
		ClusterType:               clusterTypeFlag,
		DomainName:                domainNameFlag,
		KubeconfigPath:            config.Kubeconfig,
		KubefirstStateStoreBucket: kubefirstStateStoreBucketName,
		KubefirstTeam:             kubefirstTeam,
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

		GitlabHost:         civo.GitlabHost,
		GitlabOwner:        cGitOwner,
		GitlabOwnerGroupID: cGitlabOwnerGroupID,
		GitlabUser:         cGitUser,

		GitOpsRepoAtlantisWebhookURL: fmt.Sprintf("https://atlantis.%s/events", domainNameFlag),
		GitOpsRepoNoHTTPSURL:         fmt.Sprintf("%s.com/%s/gitops.git", cGitHost, cGitOwner),
		ClusterId:                    clusterId,
	}

	viper.Set(fmt.Sprintf("%s.atlantis.webhook.url", config.GitProvider), fmt.Sprintf("https://atlantis.%s/events", domainNameFlag))
	viper.WriteConfig()

	if useTelemetryFlag {
		gitopsDirectoryTokens.UseTelemetry = "true"

		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, civo.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricInitStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	} else {
		gitopsDirectoryTokens.UseTelemetry = "false"
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

	executionControl := viper.GetBool("kubefirst-checks.cloud-credentials")
	if !executionControl {
		if os.Getenv("CIVO_TOKEN") == "" {
			fmt.Println("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security to retrieve your token\nand enter it here, then press Enter:")
			civoToken, err := term.ReadPassword(0)
			if err != nil {
				return errors.New("error reading password input from user")
			}

			os.Setenv("CIVO_TOKEN", string(civoToken))
			log.Info().Msg("CIVO_TOKEN set - continuing")
		}
		viper.Set("kubefirst-checks.cloud-credentials", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already completed cloud credentials check - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	executionControl = viper.GetBool("kubefirst-checks.state-store-creds")
	if !executionControl {
		creds, err := civo.GetAccessCredentials(kubefirstStateStoreBucketName, cloudRegionFlag)
		if err != nil {
			log.Info().Msg(err.Error())
		}

		// Verify all credentials fields are present
		var civoCredsFailureMessage string
		switch {
		case creds.AccessKeyID == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, AccessKeyID was empty - please retry your cluster creation"
		case creds.ID == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, ID was empty - please retry your cluster creation"
		case creds.Name == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, Name was empty - please retry your cluster creation"
		case creds.SecretAccessKeyID == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, SecretAccessKeyID was empty - please retry your cluster creation"
		}
		if civoCredsFailureMessage != "" {
			// Creds failed to properly parse, so remove them
			err := civo.DeleteAccessCredentials(kubefirstStateStoreBucketName, cloudRegionFlag)
			if err != nil {
				return err
			}

			// Return error
			return fmt.Errorf(civoCredsFailureMessage)
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
		// domain id
		domainId, err := civo.GetDNSInfo(domainNameFlag, cloudRegionFlag)
		if err != nil {
			log.Info().Msg(err.Error())
		}

		// viper values set in above function
		log.Info().Msgf("domainId: %s", domainId)
		domainLiveness := civo.TestDomainLiveness(false, domainNameFlag, domainId, cloudRegionFlag)
		if !domainLiveness {
			msg := "failed to check the liveness of the Domain. A valid public Domain on the same CIVO " +
				"account as the one where Kubefirst will be installed is required for this operation to " +
				"complete.\nTroubleshoot Steps:\n\n - Make sure you are using the correct CIVO account and " +
				"region.\n - Verify that you have the necessary permissions to access the domain.\n - Check " +
				"that the domain is correctly configured and is a public domain\n - Check if the " +
				"domain exists and has the correct name and domain.\n - If you don't have a Domain," +
				"please follow these instructions to create one: " +
				"https://www.civo.com/learn/configure-dns \n\n" +
				"if you are still facing issues please reach out to support team for further assistance"

			return errors.New(msg)
		}
		viper.Set("kubefirst-checks.domain-liveness", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("domain check already complete - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	executionControl = viper.GetBool("kubefirst-checks.state-store-create")
	if !executionControl {
		accessKeyId := viper.GetString("kubefirst.state-store-creds.access-key-id")
		log.Info().Msgf("access key id %s", accessKeyId)

		bucket, err := civo.CreateStorageBucket(accessKeyId, kubefirstStateStoreBucketName, cloudRegionFlag)
		if err != nil {
			log.Info().Msg(err.Error())
			return err
		}

		viper.Set("kubefirst.state-store.id", bucket.ID)
		viper.Set("kubefirst.state-store.name", bucket.Name)
		viper.Set("kubefirst-checks.state-store-create", true)
		viper.WriteConfig()
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
		return errors.New("at least one of your Civo quotas is close to its limit. Please check the error message above for additional details")
	case quotaWarnings > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
	}
	//* CIVO END

	// Objects to check for
	newRepositoryNames := []string{"gitops", "metaphor"}
	newTeamNames := []string{"admins", "developers"}

	executionControl = viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider))
	if !executionControl {
		if len(cGitToken) == 0 {
			return fmt.Errorf(
				"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init",
				strings.ToUpper(config.GitProvider),
			)
		}

		switch config.GitProvider {
		case "github":
			githubSession := github.New(cGitToken)
			newRepositoryExists := false
			// todo hoist to globals
			errorMsg := "the following repositories must be removed before continuing with your kubefirst installation.\n\t"

			for _, repositoryName := range newRepositoryNames {
				responseStatusCode := githubSession.CheckRepoExists(githubOrgFlag, repositoryName)

				// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository
				repositoryExistsStatusCode := 200
				repositoryDoesNotExistStatusCode := 404

				if responseStatusCode == repositoryExistsStatusCode {
					log.Info().Msgf("repository https://github.com/%s/%s exists", githubOrgFlag, repositoryName)
					errorMsg = errorMsg + fmt.Sprintf("https://github.com/%s/%s\n\t", githubOrgFlag, repositoryName)
					newRepositoryExists = true
				} else if responseStatusCode == repositoryDoesNotExistStatusCode {
					log.Info().Msgf("repository https://github.com/%s/%s does not exist, continuing", githubOrgFlag, repositoryName)
				}
			}
			if newRepositoryExists {
				return errors.New(errorMsg)
			}

			newTeamExists := false
			errorMsg = "the following teams must be removed before continuing with your kubefirst installation.\n\t"

			for _, teamName := range newTeamNames {
				responseStatusCode := githubSession.CheckTeamExists(githubOrgFlag, teamName)

				// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#get-a-team-by-name
				teamExistsStatusCode := 200
				teamDoesNotExistStatusCode := 404

				if responseStatusCode == teamExistsStatusCode {
					log.Info().Msgf("team https://github.com/%s/%s exists", githubOrgFlag, teamName)
					errorMsg = errorMsg + fmt.Sprintf("https://github.com/orgs/%s/teams/%s\n\t", githubOrgFlag, teamName)
					newTeamExists = true
				} else if responseStatusCode == teamDoesNotExistStatusCode {
					log.Info().Msgf("https://github.com/orgs/%s/teams/%s does not exist, continuing", githubOrgFlag, teamName)
				}
			}
			if newTeamExists {
				return errors.New(errorMsg)
			}
		case "gitlab":
			gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
			if err != nil {
				return err
			}

			// Check for existing base projects
			projects, err := gitlabClient.GetProjects()
			if err != nil {
				log.Fatal().Msgf("couldn't get gitlab projects: %s", err)
			}
			for _, repositoryName := range newRepositoryNames {
				for _, project := range projects {
					if project.Name == repositoryName {
						return fmt.Errorf("project %s already exists and will need to be deleted before continuing", repositoryName)
					}
				}
			}

			// Check for existing base projects
			// Save for detokenize
			cGitlabOwnerGroupID = gitlabClient.ParentGroupID
			viper.Set("flags.gitlab-owner-group-id", cGitlabOwnerGroupID)
			viper.WriteConfig()
			subgroups, err := gitlabClient.GetSubGroups()
			if err != nil {
				log.Fatal().Msgf("couldn't get gitlab subgroups for group %s: %s", cGitOwner, err)
			}
			for _, teamName := range newRepositoryNames {
				for _, sg := range subgroups {
					if sg.Name == teamName {
						return fmt.Errorf("subgroup %s already exists and will need to be deleted before continuing", teamName)
					}
				}
			}
		}

		viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider), true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg(fmt.Sprintf("already completed %s checks - continuing", config.GitProvider))
		progressPrinter.IncrementTracker("preflight-checks", 1)
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
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already setup kbot user - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	if useTelemetryFlag {
		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, civo.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricInitCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, civo.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricMgmtClusterInstallStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	}
	publicKeys, err := ssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	//* download dependencies to `$HOME/.k1/tools`
	progressPrinter.AddTracker("downloading-tools", "Downloading tools", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := civo.DownloadTools(
			config.KubectlClient,
			civo.KubectlClientVersion,
			civo.LocalhostOS,
			civo.LocalhostArch,
			civo.TerraformClientVersion,
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
	metaphorDirectoryTokens := civo.MetaphorTokenValues{
		ClusterName:                   clusterNameFlag,
		CloudRegion:                   cloudRegionFlag,
		ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", containerRegistryHost, cGitOwner),
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
		gitopsDirectoryTokens.GitOpsRepoGitURL = destinationGitopsRepoGitURL

		err := civo.PrepareGitRepositories(
			config.GitProvider,
			clusterNameFlag,
			clusterTypeFlag,
			destinationGitopsRepoGitURL,
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			destinationMetaphorRepoGitURL,
			config.K1Dir,
			&gitopsDirectoryTokens,
			config.MetaphorDir,
			&metaphorDirectoryTokens,
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
			log.Info().Msg("Creating github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
			err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
			if err != nil {
				return fmt.Errorf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
			}

			log.Info().Msgf("Created git repositories and teams for github.com/%s", cGitOwner)
			viper.Set("kubefirst-checks.terraform-apply-github", true)
			viper.WriteConfig()
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		} else {
			log.Info().Msg("already created github terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	case "gitlab":
		// //* create teams and repositories in gitlab
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-gitlab")
		if !executionControl {
			log.Info().Msg("Creating gitlab resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs = civo.GetGitlabTerraformEnvs(tfEnvs, cGitlabOwnerGroupID)
			err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
			if err != nil {
				return fmt.Errorf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
			}

			log.Info().Msgf("created git projects and groups for gitlab.com/%s", gitlabGroupFlag)
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
			viper.Set("kubefirst-checks.terraform-apply-gitlab", true)
			viper.WriteConfig()
		} else {
			log.Info().Msg("already created gitlab terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	}

	//* push detokenized gitops-template repository content to new remote
	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %s", destinationGitopsRepoGitURL)
	log.Info().Msgf("referencing metaphor repository: %s", destinationMetaphorRepoGitURL)

	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		// For GitLab, we currently need to add an ssh key to the authenticating user
		if config.GitProvider == "gitlab" {
			gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
			if err != nil {
				return err
			}
			keys, err := gitlabClient.GetUserSSHKeys()
			if err != nil {
				log.Fatal().Msgf("unable to check for ssh keys in gitlab: %s", err.Error())
			}

			var keyName = "kbot-ssh-key"
			var keyFound bool = false
			for _, key := range keys {
				if key.Title == keyName {
					if strings.Contains(key.Key, strings.TrimSuffix(viper.GetString("kbot.public-key"), "\n")) {
						log.Info().Msgf("ssh key %s already exists and key is up to date, continuing", keyName)
						keyFound = true
					} else {
						log.Fatal().Msgf("ssh key %s already exists and key data has drifted - please remove before continuing", keyName)
					}
				}
			}
			if !keyFound {
				log.Info().Msgf("creating ssh key %s...", keyName)
				err := gitlabClient.AddUserSSHKey(keyName, viper.GetString("kbot.public-key"))
				if err != nil {
					log.Fatal().Msgf("error adding ssh key %s: %s", keyName, err.Error())
				}
				viper.Set("kbot.gitlab-user-based-ssh-key-title", keyName)
				viper.WriteConfig()
			}
		}

		// Push gitops repo to remote
		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: config.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s: %s", destinationGitopsRepoGitURL, err)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       publicKeys,
			},
		)
		if err != nil {
			log.Panic().Msgf("error pushing detokenized metaphor repository to remote %s: %s", destinationMetaphorRepoGitURL, err)
		}

		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("successfully pushed gitops to git@%s/%s/gitops", cGitHost, cGitOwner)
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)
	}

	//* create civo cloud resources
	progressPrinter.AddTracker("applying-civo-terraform", "Applying Civo Terraform", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		log.Info().Msg("Creating civo cloud resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return fmt.Errorf("error creating civo resources with terraform %s : %s", tfEntrypoint, err)
		}

		log.Info().Msg("Created civo cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-civo", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("applying-civo-terraform", 1)
	} else {
		log.Info().Msg("already created github terraform resources")
		progressPrinter.IncrementTracker("applying-civo-terraform", 1)
	}

	clientset, err := k8s.GetClientSet(dryRunFlag, config.Kubeconfig)
	if err != nil {
		return err
	}

	// Civo Readiness checks
	progressPrinter.AddTracker("verifying-civo-cluster-readiness", "Verifying Kubernetes cluster is ready", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

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
	progressPrinter.IncrementTracker("verifying-civo-cluster-readiness", 1)

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	progressPrinter.AddTracker("bootstrapping-kubernetes-resources", "Bootstrapping Kubernetes resources", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := civo.BootstrapCivoMgmtCluster(dryRunFlag, config.Kubeconfig, config.GitProvider, cGitUser)
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

	// Handle secret creation for buildkit
	createTokensFor := []string{"metaphor"}
	switch config.GitProvider {
	// GitHub docker auth secret
	// Buildkit requires a specific format for Docker auth created as a secret
	// For GitHub, this becomes the provided token (pat)
	case "github":
		usernamePasswordString := fmt.Sprintf("%s:%s", cGitUser, cGitToken)
		usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"username": "%s", "password": "%s", "email": "%s", "auth": "%s"}}}`, containerRegistryHost, viper.GetString("flags.github-owner"), cGitToken, "k-bot@example.com", usernamePasswordStringB64)

		for _, repository := range createTokensFor {
			// Create argo workflows pull secret
			// This is formatted to work with buildkit
			argoDeployTokenSecret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-deploy", repository), Namespace: "argo"},
				Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
				Type:       "Opaque",
			}
			err = k8s.CreateSecretV2(clientset, argoDeployTokenSecret)
			if err != nil {
				log.Error().Msgf("error while creating secret for repository deploy token: %s", err)
			}
		}
	// GitLab Deploy Tokens
	// Project deploy tokens are generated for each member of createTokensForProjects
	// These deploy tokens are used to authorize against the GitLab container registry
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
		if err != nil {
			return err
		}

		for _, project := range createTokensFor {
			var p = gitlab.DeployTokenCreateParameters{
				Name:     fmt.Sprintf("%s-deploy", project),
				Username: fmt.Sprintf("%s-deploy", project),
				Scopes:   []string{"read_registry", "write_registry"},
			}

			log.Info().Msgf("creating project deploy token for project %s...", project)
			token, err := gitlabClient.CreateProjectDeployToken(project, &p)
			if err != nil {
				log.Fatal().Msgf("error creating project deploy token for project %s: %s", project, err)
			}

			if token != "" {
				log.Info().Msgf("creating secret for project deploy token for project %s...", project)
				usernamePasswordString := fmt.Sprintf("%s:%s", p.Username, token)
				usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
				dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"username": "%s", "password": "%s", "email": "%s", "auth": "%s"}}}`, containerRegistryHost, p.Username, token, "k-bot@example.com", usernamePasswordStringB64)

				createInNamespace := []string{"development", "staging", "production"}
				for _, namespace := range createInNamespace {
					deployTokenSecret := &v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-deploy", project), Namespace: namespace},
						Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
						Type:       "kubernetes.io/dockerconfigjson",
					}
					err = k8s.CreateSecretV2(clientset, deployTokenSecret)
					if err != nil {
						log.Error().Msgf("error while creating secret for project deploy token: %s", err)
					}
				}

				// Create argo workflows pull secret
				// This is formatted to work with buildkit
				argoDeployTokenSecret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-deploy", project), Namespace: "argo"},
					Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
					Type:       "Opaque",
				}
				err = k8s.CreateSecretV2(clientset, argoDeployTokenSecret)
				if err != nil {
					log.Error().Msgf("error while creating secret for project deploy token: %s", err)
				}
			}
		}
	}
	progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)

	progressPrinter.AddTracker("installing-argo-cd", "Installing and configuring ArgoCD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	//* install argocd
	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
		log.Info().Msgf("installing argocd")
		argoCDYamlPath := fmt.Sprintf("%s/registry/%s/components/argocd", config.GitopsDir, clusterNameFlag)
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClient, "--kubeconfig", config.Kubeconfig, "apply", "-k", argoCDYamlPath, "--wait")
		if err != nil {
			log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", argoCDYamlPath, err.Error())
			return err
		}
		viper.Set("kubefirst-checks.argocd-install", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argo cd already installed, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	}

	// Wait for ArgoCD to be ready
	_, err = k8s.VerifyArgoCDReadiness(clientset, true)
	if err != nil {
		log.Error().Msgf("error waiting for ArgoCD to become ready: %s", err)
		return err
	}

	restConfig, err := k8s.GetClientConfig(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	//* ArgoCD port-forward
	argoCDStopChannel := make(chan struct{}, 1)
	defer func() {
		close(argoCDStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		clientset,
		restConfig,
		"argocd-server", // todo fix this, it should `argocd
		"argocd",
		8080,
		8080,
		argoCDStopChannel,
	)
	log.Info().Msgf("port-forward to argocd is available at %s", civo.ArgocdPortForwardURL)

	//* argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	if !executionControl {
		log.Info().Msg("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

		argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
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

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		log.Info().Msg("applying the registry application to argocd")
		registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", config.K1Dir, clusterNameFlag)
		_, err := pkg.ExecShellReturnStringsV2(config.KubectlClient, "--kubeconfig", config.Kubeconfig, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
		if err != nil {
			log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
			return err
		}
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd", 1)
	}

	// Wait for Vault StatefulSet Pods to transition to Running
	progressPrinter.AddTracker("configuring-vault", "Configuring Vault", 4)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		clientset,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		120,
	)
	if err != nil {
		log.Error().Msgf("Error finding Vault StatefulSet: %s", err)
		return err
	}
	_, err = k8s.WaitForStatefulSetReady(clientset, vaultStatefulSet, 120, true)
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
		vaultClient := &vault.Conf

		// Initialize and unseal Vault
		err := vaultClient.UnsealRaftLeader(clientset, restConfig, config.Kubeconfig)
		if err != nil {
			return err
		}

		time.Sleep(time.Second * 5)
		err = vaultClient.UnsealRaftFollowers(clientset, restConfig, config.Kubeconfig)
		if err != nil {
			return err
		}

		viper.Set("kubefirst-checks.vault-initialized", true)
		viper.WriteConfig()
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
		clientset,
		restConfig,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

		//* run vault terraform
		log.Info().Msg("configuring vault with terraform")

		tfEnvs := map[string]string{}

		tfEnvs = civo.GetVaultTerraformEnvs(clientset, config, tfEnvs)
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEntrypoint := config.GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}

		log.Info().Msg("vault terraform executed successfully")
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
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
		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetUsersTerraformEnvs(clientset, config, tfEnvs)
		tfEntrypoint := config.GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}
		log.Info().Msg("executed users terraform successfully")
		// progressPrinter.IncrementTracker("step-users", 1)
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
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
		60,
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
	log.Info().Msg("welcome to your new kubefirst platform powered by Civo cloud")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	if useTelemetryFlag {
		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, civo.CloudProvider, clusterId, clusterTypeFlag, domainNameFlag, gitProviderFlag, kubefirstTeam, pkg.MetricMgmtClusterInstallCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	}

	// Set flags used to track status of active options
	helpers.SetCompletionFlags(civo.CloudProvider, config.GitProvider)

	// this is probably going to get streamlined later, but this is necessary now
	reports.CivoHandoffScreen(viper.GetString("components.argocd.password"), clusterNameFlag, domainNameFlag, cGitOwner, config, dryRunFlag, false)

	time.Sleep(time.Second * 1) // allows progress bars to finish

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return nil
}
