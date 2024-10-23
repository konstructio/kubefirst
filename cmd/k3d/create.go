/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/atotto/clipboard"
	"github.com/dustin/go-humanize"
	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/konstructio/kubefirst-api/pkg/argocd"
	"github.com/konstructio/kubefirst-api/pkg/configs"
	constants "github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst-api/pkg/gitClient"
	"github.com/konstructio/kubefirst-api/pkg/github"
	"github.com/konstructio/kubefirst-api/pkg/gitlab"
	"github.com/konstructio/kubefirst-api/pkg/handlers"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	"github.com/konstructio/kubefirst-api/pkg/progressPrinter"
	"github.com/konstructio/kubefirst-api/pkg/reports"
	"github.com/konstructio/kubefirst-api/pkg/services"
	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	"github.com/konstructio/kubefirst-api/pkg/terraform"
	"github.com/konstructio/kubefirst-api/pkg/types"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst-api/pkg/wrappers"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/segment"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

//nolint:gocyclo // this function is complex and needs to be refactored
func runK3d(cmd *cobra.Command, _ []string) error {
	ciFlag, err := cmd.Flags().GetBool("ci")
	if err != nil {
		return fmt.Errorf("failed to get 'ci' flag: %w", err)
	}

	clusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		return fmt.Errorf("failed to get 'cluster-name' flag: %w", err)
	}

	clusterTypeFlag, err := cmd.Flags().GetString("cluster-type")
	if err != nil {
		return fmt.Errorf("failed to get 'cluster-type' flag: %w", err)
	}

	githubOrgFlag, err := cmd.Flags().GetString("github-org")
	if err != nil {
		return fmt.Errorf("failed to get 'github-org' flag: %w", err)
	}

	githubUserFlag, err := cmd.Flags().GetString("github-user")
	if err != nil {
		return fmt.Errorf("failed to get 'github-user' flag: %w", err)
	}

	gitlabGroupFlag, err := cmd.Flags().GetString("gitlab-group")
	if err != nil {
		return fmt.Errorf("failed to get 'gitlab-group' flag: %w", err)
	}

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		return fmt.Errorf("failed to get 'git-provider' flag: %w", err)
	}

	gitProtocolFlag, err := cmd.Flags().GetString("git-protocol")
	if err != nil {
		return fmt.Errorf("failed to get 'git-protocol' flag: %w", err)
	}

	gitopsTemplateURLFlag, err := cmd.Flags().GetString("gitops-template-url")
	if err != nil {
		return fmt.Errorf("failed to get 'gitops-template-url' flag: %w", err)
	}

	gitopsTemplateBranchFlag, err := cmd.Flags().GetString("gitops-template-branch")
	if err != nil {
		return fmt.Errorf("failed to get 'gitops-template-branch' flag: %w", err)
	}

	installCatalogAppsFlag, err := cmd.Flags().GetString("install-catalog-apps")
	if err != nil {
		return fmt.Errorf("failed to get 'install-catalog-apps' flag: %w", err)
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return fmt.Errorf("failed to get 'use-telemetry' flag: %w", err)
	}

	utilities.CreateK1ClusterDirectory(clusterNameFlag)
	utils.DisplayLogHints()

	isValid, catalogApps, err := catalog.ValidateCatalogApps(installCatalogAppsFlag)
	if err != nil {
		return fmt.Errorf("failed to validate catalog apps: %w", err)
	}

	if !isValid {
		return errors.New("catalog apps validation failed")
	}

	switch gitProviderFlag {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("Host key for github.com: %q", key.Type())
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("Host key for gitlab.com: %q", key.Type())
	}

	if githubOrgFlag != "" && githubUserFlag != "" {
		return errors.New("only one of --github-user or --github-org can be supplied")
	}

	err = k8s.CheckForExistingPortForwards(8080, 8200, 9000, 9094)
	if err != nil {
		return fmt.Errorf("error checking existing port forwards: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpClient := http.DefaultClient

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", k3d.DomainName)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.git-protocol", gitProtocolFlag)
	viper.Set("kubefirst.cloud-provider", "k3d")
	viper.WriteConfig()

	var cGitHost, cGitOwner, cGitUser, cGitToken, containerRegistryHost string
	var cGitlabOwnerGroupID int
	switch gitProviderFlag {
	case "github":
		cGitHost = k3d.GithubHost
		containerRegistryHost = "ghcr.io"

		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		var existingToken string
		if os.Getenv("GITHUB_TOKEN") != "" {
			existingToken = os.Getenv("GITHUB_TOKEN")
		} else if viper.GetString("github.session_token") != "" {
			existingToken = viper.GetString("github.session_token")
		}
		gitHubAccessToken, err := wrappers.AuthenticateGitHubUserWrapper(existingToken, gitHubHandler)
		if err != nil {
			log.Warn().Msg(err.Error())
		}

		cGitToken = gitHubAccessToken

		err = github.VerifyTokenPermissions(cGitToken)
		if err != nil {
			return fmt.Errorf("failed to verify GitHub token permissions: %w", err)
		}

		log.Info().Msg("verifying GitHub authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(cGitToken)
		if err != nil {
			return fmt.Errorf("failed to get GitHub user: %w", err)
		}

		if githubOrgFlag != "" {
			cGitOwner = githubOrgFlag
		} else {
			cGitOwner = githubUser
		}
		cGitUser = githubUser

		viper.Set("flags.github-owner", cGitOwner)
		viper.Set("github.session_token", cGitToken)
		viper.WriteConfig()
	case "gitlab":
		if gitlabGroupFlag == "" {
			return errors.New("please provide a gitlab group using the --gitlab-group flag")
		}

		cGitToken = os.Getenv("GITLAB_TOKEN")
		if cGitToken == "" {
			return errors.New("GITLAB_TOKEN environment variable unset - please set it and try again")
		}

		err = gitlab.VerifyTokenPermissions(cGitToken)
		if err != nil {
			return fmt.Errorf("failed to verify GitLab token permissions: %w", err)
		}

		gitlabClient, err := gitlab.NewGitLabClient(cGitToken, gitlabGroupFlag)
		if err != nil {
			return fmt.Errorf("failed to create GitLab client: %w", err)
		}

		cGitHost = k3d.GitlabHost
		cGitOwner = gitlabClient.ParentGroupPath
		cGitlabOwnerGroupID = gitlabClient.ParentGroupID
		log.Info().Msgf("set gitlab owner to %q", cGitOwner)

		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set: %w", err)
		}
		cGitUser = user.Username
		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
		viper.Set("flags.gitlab-owner-group-id", cGitlabOwnerGroupID)
		viper.Set("gitlab.session_token", cGitToken)
		viper.WriteConfig()
	default:
		return fmt.Errorf("invalid git provider option %q", gitProviderFlag)
	}

	var gitDestDescriptor string
	switch gitProviderFlag {
	case "github":
		if githubOrgFlag != "" {
			gitDestDescriptor = "Organization"
		} else {
			gitDestDescriptor = "User"
		}
	case "gitlab":
		gitDestDescriptor = "Group"
	}

	config, err := k3d.GetConfig(clusterNameFlag, gitProviderFlag, cGitOwner, gitProtocolFlag)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	switch gitProviderFlag {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterID := viper.GetString("kubefirst.cluster-id")
	if clusterID == "" {
		clusterID = utils.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterID)
		viper.WriteConfig()
	}

	segClient, err := segment.InitClient(clusterID, clusterTypeFlag, gitProviderFlag)
	if err != nil {
		return fmt.Errorf("failed to initialize segment client: %w", err)
	}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 5)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	progressPrinter.IncrementTracker("preflight-checks")

	switch configs.K1Version {
	case "development":
		if strings.Contains(gitopsTemplateURLFlag, "https://github.com/konstructio/gitops-template.git") && gitopsTemplateBranchFlag == "" {
			gitopsTemplateBranchFlag = "main"
		}
	default:
		switch gitopsTemplateURLFlag {
		case "https://github.com/konstructio/gitops-template.git":
			if gitopsTemplateBranchFlag == "" {
				gitopsTemplateBranchFlag = configs.K1Version
			}
		case "https://github.com/konstructio/gitops-template":
			if gitopsTemplateBranchFlag == "" {
				gitopsTemplateBranchFlag = configs.K1Version
			}
		default:
			if gitopsTemplateBranchFlag == "" {
				return errors.New("must supply gitops-template-branch flag when gitops-template-url is overridden")
			}
		}
	}

	log.Info().Msgf("kubefirst version configs.K1Version: %q", configs.K1Version)
	log.Info().Msgf("cloning gitops-template repo url: %q", gitopsTemplateURLFlag)
	log.Info().Msgf("cloning gitops-template repo branch: %q", gitopsTemplateBranchFlag)

	atlantisWebhookSecret := viper.GetString("secrets.atlantis-webhook")
	if atlantisWebhookSecret == "" {
		atlantisWebhookSecret = utils.Random(20)
		viper.Set("secrets.atlantis-webhook", atlantisWebhookSecret)
		viper.WriteConfig()
	}

	atlantisNgrokAuthtoken := viper.GetString("secrets.atlantis-ngrok-authtoken")
	if atlantisNgrokAuthtoken == "" {
		atlantisNgrokAuthtoken = os.Getenv("NGROK_AUTHTOKEN")
		viper.Set("secrets.atlantis-ngrok-authtoken", atlantisNgrokAuthtoken)
		viper.WriteConfig()
	}

	log.Info().Msg("checking authentication to required providers")

	free, err := utils.GetAvailableDiskSize()
	if err != nil {
		return fmt.Errorf("failed to get available disk size: %w", err)
	}

	availableDiskSize := float64(free) / humanize.GByte
	if availableDiskSize < constants.MinimumAvailableDiskSize {
		return fmt.Errorf(
			"there is not enough space to proceed with the installation, a minimum of %d GB is required to proceed",
			constants.MinimumAvailableDiskSize,
		)
	}
	progressPrinter.IncrementTracker("preflight-checks")

	newRepositoryNames := []string{"gitops", "metaphor"}
	newTeamNames := []string{"admins", "developers"}

	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider))
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.GitCredentialsCheckStarted, "")
		if len(cGitToken) == 0 {
			msg := fmt.Sprintf("please set a %s_TOKEN environment variable to continue", strings.ToUpper(config.GitProvider))
			telemetry.SendEvent(segClient, telemetry.GitCredentialsCheckFailed, msg)
			return errors.New(msg)
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
			return fmt.Errorf("failed to initialize Git provider: %w", err)
		}

		viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider), true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.GitCredentialsCheckCompleted, "")
		progressPrinter.IncrementTracker("preflight-checks")
	} else {
		log.Info().Msg(fmt.Sprintf("already completed %q checks - continuing", config.GitProvider))
		progressPrinter.IncrementTracker("preflight-checks")
	}

	var gitopsRepoURL string
	executionControl = viper.GetBool("kubefirst-checks.kbot-setup")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.KbotSetupStarted, "")

		log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
		sshPrivateKey, sshPublicKey, err = utils.CreateSSHKeyPair()
		if err != nil {
			telemetry.SendEvent(segClient, telemetry.KbotSetupFailed, err.Error())
			return fmt.Errorf("failed to create SSH key pair: %w", err)
		}
		log.Info().Msg("ssh key pair creation complete")

		viper.Set("kbot.private-key", sshPrivateKey)
		viper.Set("kbot.public-key", sshPublicKey)
		viper.Set("kbot.username", "kbot")
		viper.Set("kubefirst-checks.kbot-setup", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.KbotSetupCompleted, "")
		log.Info().Msg("kbot-setup complete")
		progressPrinter.IncrementTracker("preflight-checks")
	} else {
		log.Info().Msg("already setup kbot user - continuing")
		progressPrinter.IncrementTracker("preflight-checks")
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	telemetry.SendEvent(segClient, telemetry.InitCompleted, "")

	switch config.GitProtocol {
	case "https":
		gitopsRepoURL = config.DestinationGitopsRepoURL
	default:
		gitopsRepoURL = config.DestinationGitopsRepoGitURL
	}

	gitopsDirectoryTokens := k3d.GitopsDirectoryValues{
		GithubOwner:                   cGitOwner,
		GithubUser:                    cGitUser,
		GitlabOwner:                   cGitOwner,
		GitlabOwnerGroupID:            cGitlabOwnerGroupID,
		GitlabUser:                    cGitUser,
		DomainName:                    k3d.DomainName,
		AtlantisAllowList:             fmt.Sprintf("%s/%s/*", cGitHost, cGitOwner),
		AlertsEmail:                   "REMOVE_THIS_VALUE",
		ClusterName:                   clusterNameFlag,
		ClusterType:                   clusterTypeFlag,
		GithubHost:                    k3d.GithubHost,
		GitlabHost:                    k3d.GitlabHost,
		ArgoWorkflowsIngressURL:       fmt.Sprintf("https://argo.%s", k3d.DomainName),
		VaultIngressURL:               fmt.Sprintf("https://vault.%s", k3d.DomainName),
		ArgocdIngressURL:              fmt.Sprintf("https://argocd.%s", k3d.DomainName),
		AtlantisIngressURL:            fmt.Sprintf("https://atlantis.%s", k3d.DomainName),
		MetaphorDevelopmentIngressURL: fmt.Sprintf("https://metaphor-development.%s", k3d.DomainName),
		MetaphorStagingIngressURL:     fmt.Sprintf("https://metaphor-staging.%s", k3d.DomainName),
		MetaphorProductionIngressURL:  fmt.Sprintf("https://metaphor-production.%s", k3d.DomainName),
		KubefirstVersion:              configs.K1Version,
		KubefirstTeam:                 kubefirstTeam,
		KubeconfigPath:                config.Kubeconfig,
		GitopsRepoURL:                 gitopsRepoURL,
		GitProvider:                   config.GitProvider,
		ClusterID:                     clusterID,
		CloudProvider:                 k3d.CloudProvider,
	}

	if useTelemetryFlag {
		gitopsDirectoryTokens.UseTelemetry = "true"
	} else {
		gitopsDirectoryTokens.UseTelemetry = "false"
	}

	httpAuth := &githttps.BasicAuth{
		Username: cGitUser,
		Password: cGitToken,
	}

	if err != nil {
		log.Info().Msgf("generate public keys failed: %q", err.Error())
		return fmt.Errorf("failed to generate public keys: %w", err)
	}

	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := k3d.DownloadTools(clusterNameFlag, config.GitProvider, cGitOwner, config.ToolsDir, config.GitProtocol)
		if err != nil {
			return fmt.Errorf("failed to download tools: %w", err)
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}
	progressPrinter.IncrementTracker("preflight-checks")

	metaphorTemplateTokens := k3d.MetaphorTokenValues{
		ClusterName:                   clusterNameFlag,
		CloudRegion:                   cloudRegionFlag,
		ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", containerRegistryHost, cGitOwner),
		DomainName:                    k3d.DomainName,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", k3d.DomainName),
		MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", k3d.DomainName),
		MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", k3d.DomainName),
	}

	progressPrinter.IncrementTracker("preflight-checks")
	progressPrinter.AddTracker("cloning-and-formatting-git-repositories", "Cloning and formatting git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	removeAtlantis := false
	if viper.GetString("secrets.atlantis-ngrok-authtoken") == "" {
		removeAtlantis = true
	}
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {
		log.Info().Msg("generating your new gitops repository")
		err := k3d.PrepareGitRepositories(
			config.GitProvider,
			clusterNameFlag,
			clusterTypeFlag,
			config.DestinationGitopsRepoURL,
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			config.DestinationMetaphorRepoURL,
			config.K1Dir,
			&gitopsDirectoryTokens,
			config.MetaphorDir,
			&metaphorTemplateTokens,
			gitProtocolFlag,
			removeAtlantis,
		)
		if err != nil {
			return fmt.Errorf("failed to prepare git repositories: %w", err)
		}

		viper.Set("kubefirst-checks.gitops-ready-to-push", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("cloning-and-formatting-git-repositories")
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
		progressPrinter.IncrementTracker("cloning-and-formatting-git-repositories")
	}

	progressPrinter.AddTracker("applying-git-terraform", fmt.Sprintf("Applying %s Terraform", config.GitProvider), 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	switch config.GitProvider {
	case "github":
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
		if !executionControl {
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitHub resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{
				"GITHUB_TOKEN":                 cGitToken,
				"GITHUB_OWNER":                 cGitOwner,
				"TF_VAR_kbot_ssh_public_key":   viper.GetString("kbot.public-key"),
				"AWS_ACCESS_KEY_ID":            constants.MinioDefaultUsername,
				"AWS_SECRET_ACCESS_KEY":        constants.MinioDefaultPassword,
				"TF_VAR_aws_access_key_id":     constants.MinioDefaultUsername,
				"TF_VAR_aws_secret_access_key": constants.MinioDefaultPassword,
			}
			if config.GitProtocol == "https" {
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
			}

			err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Errorf("error creating GitHub resources with terraform %q: %w", tfEntrypoint, err)
				telemetry.SendEvent(segClient, telemetry.GitTerraformApplyFailed, msg.Error())
				return msg
			}

			log.Info().Msgf("created git repositories for github.com/%s", cGitOwner)
			viper.Set("kubefirst-checks.terraform-apply-github", true)
			viper.WriteConfig()
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyCompleted, "")
			progressPrinter.IncrementTracker("applying-git-terraform")
		} else {
			log.Info().Msg("already created GitHub Terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform")
		}
	case "gitlab":
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-gitlab")
		if !executionControl {
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitLab resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{
				"GITLAB_TOKEN":                 cGitToken,
				"GITLAB_OWNER":                 gitlabGroupFlag,
				"TF_VAR_owner_group_id":        strconv.Itoa(cGitlabOwnerGroupID),
				"TF_VAR_kbot_ssh_public_key":   viper.GetString("kbot.public-key"),
				"AWS_ACCESS_KEY_ID":            constants.MinioDefaultUsername,
				"AWS_SECRET_ACCESS_KEY":        constants.MinioDefaultPassword,
				"TF_VAR_aws_access_key_id":     constants.MinioDefaultUsername,
				"TF_VAR_aws_secret_access_key": constants.MinioDefaultPassword,
			}
			if config.GitProtocol == "https" {
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
			}

			err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Errorf("error creating GitLab resources with terraform %q: %w", tfEntrypoint, err)
				telemetry.SendEvent(segClient, telemetry.GitTerraformApplyFailed, msg.Error())
				return msg
			}

			log.Info().Msgf("created git projects and groups for gitlab.com/%s", gitlabGroupFlag)
			viper.Set("kubefirst-checks.terraform-apply-gitlab", true)
			viper.WriteConfig()
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyCompleted, "")
			progressPrinter.IncrementTracker("applying-git-terraform")
		} else {
			log.Info().Msg("already created GitLab Terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform")
		}
	}

	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %q", config.DestinationGitopsRepoGitURL)
	log.Info().Msgf("referencing metaphor repository: %q", config.DestinationMetaphorRepoURL)

	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.GitopsRepoPushStarted, "")

		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			return fmt.Errorf("error opening repo at %q: %w", config.GitopsDir, err)
		}

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			return fmt.Errorf("error opening repo at %q: %w", config.MetaphorDir, err)
		}

		err = utils.EvalSSHKey(&types.EvalSSHKeyRequest{
			GitProvider:     gitProviderFlag,
			GitlabGroupFlag: gitlabGroupFlag,
			GitToken:        cGitToken,
		})
		if err != nil {
			return fmt.Errorf("failed to evaluate SSH key: %w", err)
		}

		err = gitopsRepo.Push(
			&git.PushOptions{
				RemoteName: config.GitProvider,
				Auth:       httpAuth,
			},
		)
		if err != nil {
			msg := fmt.Errorf("error pushing detokenized gitops repository to remote %q: %w", config.DestinationGitopsRepoGitURL, err)
			telemetry.SendEvent(segClient, telemetry.GitopsRepoPushFailed, msg.Error())
			if !strings.Contains(msg.Error(), "already up-to-date") {
				log.Print(msg.Error())
				return msg
			}
		}

		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       httpAuth,
			},
		)
		if err != nil {
			msg := fmt.Errorf("error pushing detokenized metaphor repository to remote %q: %w", config.DestinationMetaphorRepoURL, err)
			telemetry.SendEvent(segClient, telemetry.GitopsRepoPushFailed, msg.Error())
			if !strings.Contains(msg.Error(), "already up-to-date") {
				return msg
			}
		}
		log.Info().Msgf("successfully pushed gitops and metaphor repositories to https://%s/%s", cGitHost, cGitOwner)

		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.GitopsRepoPushCompleted, "")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream")
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream")
	}

	progressPrinter.AddTracker("creating-k3d-cluster", "Creating k3d cluster", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		telemetry.SendEvent(segClient, telemetry.CloudTerraformApplyStarted, "")

		log.Info().Msg("Creating k3d cluster")

		err := k3d.ClusterCreate(clusterNameFlag, config.K1Dir, config.K3dClient, config.Kubeconfig)
		if err != nil {
			msg := fmt.Errorf("error creating k3d resources with k3d client %q: %w", config.K3dClient, err)
			viper.Set("kubefirst-checks.create-k3d-cluster-failed", true)
			viper.WriteConfig()
			telemetry.SendEvent(segClient, telemetry.CloudTerraformApplyFailed, msg.Error())
			return msg
		}

		log.Info().Msg("successfully created k3d cluster")
		viper.Set("kubefirst-checks.create-k3d-cluster", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.CloudTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("creating-k3d-cluster")
	} else {
		log.Info().Msg("already created k3d cluster resources")
		progressPrinter.IncrementTracker("creating-k3d-cluster")
	}

	kcfg, err := k8s.CreateKubeConfig(false, config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	progressPrinter.AddTracker("bootstrapping-kubernetes-resources", "Bootstrapping Kubernetes resources", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := k3d.GenerateTLSSecrets(kcfg.Clientset, *config)
		if err != nil {
			return fmt.Errorf("failed to generate TLS secrets: %w", err)
		}

		err = k3d.AddK3DSecrets(
			gitopsRepoURL,
			viper.GetString("kbot.private-key"),
			config.GitProvider,
			cGitUser,
			config.Kubeconfig,
			cGitToken,
		)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return fmt.Errorf("failed to add Kubernetes secrets: %w", err)
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources")
	} else {
		log.Info().Msg("already added secrets to k3d cluster")
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources")
	}

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
		return fmt.Errorf("failed to create container registry secret: %w", err)
	}
	progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources")

	progressPrinter.AddTracker("verifying-k3d-cluster-readiness", "Verifying Kubernetes cluster is ready", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	traefikDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"traefik",
		"kube-system",
		240,
	)
	if err != nil {
		return fmt.Errorf("error finding traefik deployment: %w", err)
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, traefikDeployment, 240)
	if err != nil {
		return fmt.Errorf("error waiting for traefik deployment ready state: %w", err)
	}
	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness")

	metricsServerDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"k8s-app",
		"metrics-server",
		"kube-system",
		240,
	)
	if err != nil {
		return fmt.Errorf("error finding metrics-server deployment: %w", err)
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, metricsServerDeployment, 240)
	if err != nil {
		return fmt.Errorf("error waiting for metrics-server deployment ready state: %w", err)
	}
	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness")

	time.Sleep(time.Second * 20)

	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness")

	progressPrinter.AddTracker("installing-argo-cd", "Installing and configuring Argo CD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	argoCDInstallPath := fmt.Sprintf("github.com:konstructio/manifests/argocd/k3d?ref=%s", constants.KubefirstManifestRepoRef)
	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.ArgoCDInstallStarted, "")

		log.Info().Msgf("installing ArgoCD")

		yamlData, err := kcfg.KustomizeBuild(argoCDInstallPath)
		if err != nil {
			return fmt.Errorf("failed to build ArgoCD manifests: %w", err)
		}

		output, err := kcfg.SplitYAMLFile(yamlData)
		if err != nil {
			return fmt.Errorf("failed to split YAML file: %w", err)
		}

		if err := kcfg.ApplyObjects(output); err != nil {
			telemetry.SendEvent(segClient, telemetry.ArgoCDInstallFailed, err.Error())
			return fmt.Errorf("failed to apply ArgoCD objects: %w", err)
		}

		viper.Set("kubefirst-checks.argocd-install", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.ArgoCDInstallCompleted, "")
		progressPrinter.IncrementTracker("installing-argo-cd")
	} else {
		log.Info().Msg("ArgoCD already installed, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd")
	}

	_, err = k8s.VerifyArgoCDReadiness(kcfg.Clientset, true, 300)
	if err != nil {
		return fmt.Errorf("error waiting for ArgoCD to become ready: %w", err)
	}

	var argocdPassword string
	executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	if !executionControl {
		log.Info().Msg("Setting ArgoCD username and password credentials")

		argocd.ArgocdSecretClient = kcfg.Clientset.CoreV1().Secrets("argocd")

		argocdPassword = k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
		if argocdPassword == "" {
			return errors.New("ArgoCD password not found in secret")
		}

		viper.Set("components.argocd.password", argocdPassword)
		viper.Set("components.argocd.username", "admin")
		viper.WriteConfig()
		log.Info().Msg("ArgoCD username and password credentials set successfully")
		log.Info().Msg("Getting an ArgoCD auth token")

		var argoCDToken string
		if err := utils.TestEndpointTLS(strings.Replace(k3d.ArgocdURL, "https://", "", 1)); err != nil {
			argoCDStopChannel := make(chan struct{}, 1)
			log.Info().Msgf("ArgoCD not available via https, using http")
			defer func() {
				close(argoCDStopChannel)
			}()
			k8s.OpenPortForwardPodWrapper(
				kcfg.Clientset,
				kcfg.RestConfig,
				"argocd-server",
				"argocd",
				8080,
				8080,
				argoCDStopChannel,
			)
			argoCDHTTPURL := strings.Replace(
				k3d.ArgocdURL,
				"https://",
				"http://",
				1,
			) + ":8080"
			argoCDToken, err = argocd.GetArgocdTokenV2(argoCDHTTPURL, "admin", argocdPassword)
			if err != nil {
				return fmt.Errorf("failed to get ArgoCD token: %w", err)
			}
		} else {
			argoCDToken, err = argocd.GetArgocdTokenV2(k3d.ArgocdURL, "admin", argocdPassword)
			if err != nil {
				return fmt.Errorf("failed to get ArgoCD token: %w", err)
			}
		}

		log.Info().Msg("ArgoCD admin auth token set")

		viper.Set("components.argocd.auth-token", argoCDToken)
		viper.Set("kubefirst-checks.argocd-credentials-set", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("installing-argo-cd")
	} else {
		log.Info().Msg("ArgoCD credentials already set, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd")
	}

	if configs.K1Version == "development" {
		err := clipboard.WriteAll(argocdPassword)
		if err != nil {
			log.Error().Err(err).Msg("failed to copy ArgoCD password to clipboard")
		}

		if os.Getenv("SKIP_ARGOCD_LAUNCH") != "true" || !ciFlag {
			err = utils.OpenBrowser(constants.ArgoCDLocalURLTLS)
			if err != nil {
				log.Error().Err(err).Msg("failed to open ArgoCD URL in browser")
			}
		}
	}

	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.CreateRegistryStarted, "")
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return fmt.Errorf("failed to create ArgoCD client: %w", err)
		}

		log.Info().Msg("applying the registry application to ArgoCD")
		registryApplicationObject := argocd.GetArgoCDApplicationObject(gitopsRepoURL, fmt.Sprintf("registry/%s", clusterNameFlag))

		err = k3d.RestartDeployment(context.Background(), kcfg.Clientset, "argocd", "argocd-applicationset-controller")
		if err != nil {
			return fmt.Errorf("error in restarting ArgoCD controller: %w", err)
		}

		err = wait.PollImmediate(5*time.Second, 20*time.Second, func() (bool, error) {
			_, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
			if err != nil {
				if errors.Is(err, syscall.ECONNREFUSED) {
					return false, nil
				}

				if apierrors.IsAlreadyExists(err) {
					return true, nil
				}

				return false, fmt.Errorf("error creating ArgoCD application: %w", err)
			}
			return true, nil
		})
		if err != nil {
			return fmt.Errorf("error creating ArgoCD application: %w", err)
		}

		log.Info().Msg("ArgoCD application created successfully")
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.CreateRegistryCompleted, "")
		progressPrinter.IncrementTracker("installing-argo-cd")
	} else {
		log.Info().Msg("ArgoCD registry create already done, continuing")
		progressPrinter.IncrementTracker("installing-argo-cd")
	}

	progressPrinter.AddTracker("configuring-vault", "Configuring Vault", 4)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		120,
	)
	if err != nil {
		return fmt.Errorf("error finding Vault StatefulSet: %w", err)
	}
	_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 120, true)
	if err != nil {
		return fmt.Errorf("error waiting for Vault StatefulSet ready state: %w", err)
	}
	progressPrinter.IncrementTracker("configuring-vault")

	time.Sleep(time.Second * 10)
	progressPrinter.IncrementTracker("configuring-vault")

	executionControl = viper.GetBool("kubefirst-checks.vault-initialized")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.VaultInitializationStarted, "")

		vaultHandlerPath := "github.com:konstructio/manifests.git/vault-handler/replicas-1"

		yamlData, err := kcfg.KustomizeBuild(vaultHandlerPath)
		if err != nil {
			return fmt.Errorf("failed to build vault handler manifests: %w", err)
		}

		output, err := kcfg.SplitYAMLFile(yamlData)
		if err != nil {
			return fmt.Errorf("failed to split YAML file: %w", err)
		}

		if err := kcfg.ApplyObjects(output); err != nil {
			return fmt.Errorf("failed to apply vault handler objects: %w", err)
		}

		job, err := k8s.ReturnJobObject(kcfg.Clientset, "vault", "vault-handler")
		if err != nil {
			return fmt.Errorf("failed to get vault job object: %w", err)
		}
		_, err = k8s.WaitForJobComplete(kcfg.Clientset, job.GetName(), job.GetNamespace(), 240)
		if err != nil {
			msg := fmt.Errorf("could not run vault unseal job: %w", err)
			telemetry.SendEvent(segClient, telemetry.VaultInitializationFailed, msg.Error())
			return msg
		}

		viper.Set("kubefirst-checks.vault-initialized", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.VaultInitializationCompleted, "")
		progressPrinter.IncrementTracker("configuring-vault")
	} else {
		log.Info().Msg("vault is already initialized - skipping")
		progressPrinter.IncrementTracker("configuring-vault")
	}

	minioStopChannel := make(chan struct{}, 1)
	defer func() {
		close(minioStopChannel)
	}()

	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		"minio",
		"minio",
		9000,
		9000,
		minioStopChannel,
	)

	minioClient, err := minio.New(constants.MinioPortForwardEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(constants.MinioDefaultUsername, constants.MinioDefaultPassword, ""),
		Secure: false,
		Region: constants.MinioRegion,
	})
	if err != nil {
		return fmt.Errorf("error creating Minio client: %w", err)
	}

	objectName := fmt.Sprintf("terraform/%s/terraform.tfstate", config.GitProvider)
	filePath := config.K1Dir + fmt.Sprintf("/gitops/%s", objectName)
	contentType := "xl.meta"
	bucketName := "kubefirst-state-store"
	log.Info().Msgf("BucketName: %q", bucketName)

	viper.Set("kubefirst.state-store.name", bucketName)
	viper.Set("kubefirst.state-store.hostname", "minio-console.kubefirst.dev")
	viper.Set("kubefirst.state-store-creds.access-key-id", constants.MinioDefaultUsername)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", constants.MinioDefaultPassword)

	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("error uploading to Minio bucket: %w", err)
	}

	log.Printf("Successfully uploaded %q to bucket %q", objectName, info.Bucket)

	progressPrinter.IncrementTracker("configuring-vault")

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

	var vaultRootToken string
	secData, err := k8s.ReadSecretV2(kcfg.Clientset, "vault", "vault-unseal-secret")
	if err != nil {
		return fmt.Errorf("failed to read vault unseal secret: %w", err)
	}

	vaultRootToken = secData["root-token"]

	kubernetesInClusterAPIService, err := k8s.ReadService(config.Kubeconfig, "default", "kubernetes")
	if err != nil {
		return fmt.Errorf("error looking up kubernetes api server service: %w", err)
	}

	if err := utils.TestEndpointTLS(strings.Replace(k3d.VaultURL, "https://", "", 1)); err != nil {
		return fmt.Errorf("unable to reach vault over https: %w", err)
	}

	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.VaultTerraformApplyStarted, "")

		tfEnvs := map[string]string{}
		var usernamePasswordString, base64DockerAuth string

		if config.GitProvider == "gitlab" {
			usernamePasswordString = fmt.Sprintf("%s:%s", "container-registry-auth", containerRegistryAuthToken)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

			tfEnvs["TF_VAR_container_registry_auth"] = containerRegistryAuthToken
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(cGitlabOwnerGroupID)
		} else {
			usernamePasswordString = fmt.Sprintf("%s:%s", cGitUser, cGitToken)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		}

		log.Info().Msg("configuring vault with terraform")

		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
		tfEnvs[fmt.Sprintf("TF_VAR_%s_user", config.GitProvider)] = cGitUser
		tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
		tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_kbot_ssh_private_key"] = viper.GetString("kbot.private-key")
		tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
		tfEnvs["TF_VAR_kubernetes_api_endpoint"] = fmt.Sprintf("https://%s", kubernetesInClusterAPIService.Spec.ClusterIP)
		tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = viper.GetString(fmt.Sprintf("flags.%s-owner", config.GitProvider))
		tfEnvs["AWS_ACCESS_KEY_ID"] = constants.MinioDefaultUsername
		tfEnvs["AWS_SECRET_ACCESS_KEY"] = constants.MinioDefaultPassword
		tfEnvs["TF_VAR_aws_access_key_id"] = constants.MinioDefaultUsername
		tfEnvs["TF_VAR_aws_secret_access_key"] = constants.MinioDefaultPassword
		tfEnvs["TF_VAR_ngrok_authtoken"] = viper.GetString("secrets.atlantis-ngrok-authtoken")

		tfEntrypoint := config.GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			telemetry.SendEvent(segClient, telemetry.VaultTerraformApplyStarted, err.Error())
			return fmt.Errorf("failed to execute vault terraform: %w", err)
		}
		log.Info().Msg("vault terraform executed successfully")
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.VaultTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("configuring-vault")
	} else {
		log.Info().Msg("already executed vault terraform")
		progressPrinter.IncrementTracker("configuring-vault")
	}

	progressPrinter.AddTracker("creating-users", "Creating users", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-users")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.UsersTerraformApplyStarted, "")

		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
		tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = cGitToken
		tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = cGitOwner

		tfEntrypoint := config.GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			telemetry.SendEvent(segClient, telemetry.UsersTerraformApplyStarted, err.Error())
			return fmt.Errorf("failed to apply users terraform: %w", err)
		}
		log.Info().Msg("executed users terraform successfully")
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.UsersTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("creating-users")
	} else {
		log.Info().Msg("already created users with terraform")
		progressPrinter.IncrementTracker("creating-users")
	}

	progressPrinter.AddTracker("wrapping-up", "Wrapping up", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.post-detokenize")
	if !executionControl {
		if err := k3d.PostRunPrepareGitopsRepository(config.GitopsDir); err != nil {
			return fmt.Errorf("error detokenizing post run: %w", err)
		}

		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			return fmt.Errorf("error opening repo at %q: %w", config.GitopsDir, err)
		}
		_, err = os.Stat(fmt.Sprintf("%s/terraform/%s/remote-backend.md", config.GitopsDir, config.GitProvider))
		if err == nil {
			err = os.Rename(fmt.Sprintf("%s/terraform/%s/remote-backend.md", config.GitopsDir, config.GitProvider), fmt.Sprintf("%s/terraform/%s/remote-backend.tf", config.GitopsDir, config.GitProvider))
			if err != nil {
				return fmt.Errorf("failed to rename remote-backend.md to remote-backend.tf: %w", err)
			}
		}

		err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content post run")
		if err != nil {
			return fmt.Errorf("failed to commit initial detokenized gitops-template repo content: %w", err)
		}
		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: config.GitProvider,
			Auth:       httpAuth,
		})
		if err != nil {
			return fmt.Errorf("failed to push initial detokenized gitops-template repo content: %w", err)
		}
		viper.Set("kubefirst-checks.post-detokenize", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already detokenized post run")
	}

	progressPrinter.IncrementTracker("wrapping-up")

	argoDeployment, err := k8s.ReturnDeploymentObject(kcfg.Clientset, "app.kubernetes.io/instance", "argo", "argo", 1200)
	if err != nil {
		return fmt.Errorf("error finding Argo Workflows Deployment: %w", err)
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, argoDeployment, 120)
	if err != nil {
		return fmt.Errorf("error waiting for Argo Workflows Deployment ready state: %w", err)
	}

	utils.SetClusterStatusFlags(k3d.CloudProvider, config.GitProvider)

	cluster := utilities.CreateClusterRecordFromRaw(useTelemetryFlag, cGitOwner, cGitUser, cGitToken, cGitlabOwnerGroupID, gitopsTemplateURLFlag, gitopsTemplateBranchFlag, catalogApps)

	err = utilities.ExportCluster(cluster, kcfg)
	if err != nil {
		log.Error().Err(err).Msg("error exporting cluster object")
		viper.Set("kubefirst.setup-complete", false)
		viper.Set("kubefirst-checks.cluster-install-complete", false)
		viper.WriteConfig()
		return fmt.Errorf("failed to export cluster object: %w", err)
	}

	kubefirstDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"kubefirst",
		"kubefirst",
		600,
	)
	if err != nil {
		return fmt.Errorf("error finding kubefirst Deployment: %w", err)
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, kubefirstDeployment, 120)
	if err != nil {
		return fmt.Errorf("error waiting for kubefirst Deployment ready state: %w", err)
	}
	progressPrinter.IncrementTracker("wrapping-up")

	err = utils.OpenBrowser(constants.KubefirstConsoleLocalURLTLS)
	if err != nil {
		log.Error().Err(err).Msg("failed to open Kubefirst console in browser")
	}

	telemetry.SendEvent(segClient, telemetry.ClusterInstallCompleted, "")
	viper.Set("kubefirst-checks.cluster-install-complete", true)
	viper.WriteConfig()

	log.Info().Msg("kubefirst installation complete")
	log.Info().Msg("welcome to your new Kubefirst platform running in K3D")
	time.Sleep(1 * time.Second)

	reports.LocalHandoffScreenV2(clusterNameFlag, gitDestDescriptor, cGitOwner, config, ciFlag)

	if ciFlag {
		progress.Progress.Quit()
	}

	return nil
}
