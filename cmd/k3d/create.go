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

	"github.com/atotto/clipboard"
	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/konstructio/kubefirst-api/pkg/argocd"
	"github.com/konstructio/kubefirst-api/pkg/configs"
	constants "github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst-api/pkg/gitClient"
	github "github.com/konstructio/kubefirst-api/pkg/github"
	gitlab "github.com/konstructio/kubefirst-api/pkg/gitlab"
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func runK3d(cmd *cobra.Command, args []string) error {
	ciFlag, err := cmd.Flags().GetBool("ci")
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

	githubOrgFlag, err := cmd.Flags().GetString("github-org")
	if err != nil {
		return err
	}

	githubUserFlag, err := cmd.Flags().GetString("github-user")
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

	installCatalogAppsFlag, err := cmd.Flags().GetString("install-catalog-apps")
	if err != nil {
		return err
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return err
	}

	// // If cluster setup is complete, return
	// clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	// if clusterSetupComplete {
	// 	return fmt.Errorf("this cluster install process has already completed successfully")
	// }

	utilities.CreateK1ClusterDirectory(clusterNameFlag)
	utils.DisplayLogHints()

	isValid, catalogApps, err := catalog.ValidateCatalogApps(installCatalogAppsFlag)
	if !isValid {
		return err
	}

	switch gitProviderFlag {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return errors.New("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "github.com", key.Type())
		}
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return errors.New("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "gitlab.com", key.Type())
		}
	}

	// Either user or org can be specified for github, not both
	if githubOrgFlag != "" && githubUserFlag != "" {
		return errors.New("only one of --github-user or --github-org can be supplied")
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(8080, 8200, 9000, 9094)
	if err != nil {
		return fmt.Errorf("%s - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing", err.Error())
	}

	// Verify Docker is running # TODO: reintroduce once we support more runtimes
	// dcli := docker.DockerClientWrapper{
	// 	Client: docker.NewDockerClient(),
	// }
	// _, err = dcli.CheckDockerReady()
	// if err != nil {
	// 	return err
	// }

	// Global context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clients
	httpClient := http.DefaultClient

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	// Store flags for application state maintenance
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", k3d.DomainName)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.git-protocol", gitProtocolFlag)
	viper.Set("kubefirst.cloud-provider", "k3d")
	viper.WriteConfig()

	// Switch based on git provider, set params
	var cGitHost, cGitOwner, cGitUser, cGitToken, containerRegistryHost string
	var cGitlabOwnerGroupID int
	switch gitProviderFlag {
	case "github":
		cGitHost = k3d.GithubHost
		containerRegistryHost = "ghcr.io"

		// Attempt to retrieve session-scoped token for GitHub user
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		//
		var existingToken string
		if os.Getenv("GITHUB_TOKEN") != "" {
			existingToken = os.Getenv("GITHUB_TOKEN")
		} else if os.Getenv("GITHUB_TOKEN") == "" && viper.GetString("github.session_token") != "" {
			existingToken = viper.GetString("github.session_token")
		}
		gitHubAccessToken, err := wrappers.AuthenticateGitHubUserWrapper(existingToken, gitHubHandler)
		if err != nil {
			log.Warn().Msgf(err.Error())
		}

		// Token will either be user-provided or generated by kubefirst invocation
		cGitToken = gitHubAccessToken

		// Verify token scopes
		err = github.VerifyTokenPermissions(cGitToken)
		if err != nil {
			return err
		}

		log.Info().Msg("verifying github authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(cGitToken)
		if err != nil {
			return err
		}

		// Owner is either an organization or a personal user's GitHub handle
		if githubOrgFlag != "" {
			cGitOwner = githubOrgFlag
		} else if githubUserFlag != "" {
			cGitOwner = githubUser
		} else if githubOrgFlag == "" && githubUserFlag == "" {
			cGitOwner = githubUser
		}
		cGitUser = githubUser

		viper.Set("flags.github-owner", cGitOwner)
		viper.Set("github.session_token", cGitToken)
		viper.WriteConfig()
	case "gitlab":
		if gitlabGroupFlag == "" {
			return fmt.Errorf("please provide a gitlab group using the --gitlab-group flag")
		}

		if os.Getenv("GITLAB_TOKEN") == "" {
			return fmt.Errorf("GITLAB_TOKEN environment variable unset - please set it and try again")
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

		cGitHost = k3d.GitlabHost
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
		viper.Set("gitlab.session_token", cGitToken)
		viper.WriteConfig()
	default:
		log.Error().Msgf("invalid git provider option")
	}

	// Ask for confirmation
	var gitDestDescriptor string
	switch gitProviderFlag {
	case "github":
		if githubOrgFlag != "" {
			gitDestDescriptor = "Organization"
		}
		if githubUserFlag != "" {
			gitDestDescriptor = "User"
		}
		if githubUserFlag == "" && githubOrgFlag == "" {
			gitDestDescriptor = "User"
		}
	case "gitlab":
		gitDestDescriptor = "Group"
	}

	// todo
	// Since it's possible to stop and restart, cGitOwner may need to be reset
	//if cGitOwner == "" {
	//	switch gitProviderFlag {
	//	case "github":
	//		cGitOwner = viper.GetString("flags.github-owner")
	//	case "gitlab":
	//		cGitOwner = viper.GetString("flags.gitlab-owner")
	//	}
	//}
	//
	//model, err := presentRecap(gitProviderFlag, gitDestDescriptor, cGitOwner)
	//if err != nil {
	//	return err
	//}
	//_, err = tea.NewProgram(model).Run()
	//if err != nil {
	//	return err
	//}

	// Instantiate K3d config
	config := k3d.GetConfig(clusterNameFlag, gitProviderFlag, cGitOwner, gitProtocolFlag)
	switch gitProviderFlag {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = utils.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}

	segClient := segment.InitClient(clusterId, clusterTypeFlag, gitProviderFlag)

	// Progress output
	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 5)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	progressPrinter.IncrementTracker("preflight-checks", 1)

	// this branch flag value is overridden with a tag when running from a
	// kubefirst binary for version compatibility
	switch configs.K1Version {
	case "development":
		if strings.Contains(gitopsTemplateURLFlag, "https://github.com/konstructio/gitops-template.git") && gitopsTemplateBranchFlag == "" {
			gitopsTemplateBranchFlag = "main"
		}
	default:
		switch gitopsTemplateURLFlag {
		case "https://github.com/konstructio/gitops-template.git": // default value
			if gitopsTemplateBranchFlag == "" {
				gitopsTemplateBranchFlag = configs.K1Version
			}
		case "https://github.com/konstructio/gitops-template": // edge case for valid but incomplete url
			if gitopsTemplateBranchFlag == "" {
				gitopsTemplateBranchFlag = configs.K1Version
			}
		default: // not equal to our defaults
			if gitopsTemplateBranchFlag == "" { // didn't supply the branch flag but they did supply the  repo flag
				return fmt.Errorf("must supply gitops-template-branch flag when gitops-template-url is overridden")
			}
		}
	}

	log.Info().Msgf("kubefirst version configs.K1Version: %s ", configs.K1Version)
	log.Info().Msgf("cloning gitops-template repo url: %s ", gitopsTemplateURLFlag)
	log.Info().Msgf("cloning gitops-template repo branch: %s ", gitopsTemplateBranchFlag)

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

	// check disk
	free, err := utils.GetAvailableDiskSize()
	if err != nil {
		return err
	}

	// convert available disk size to GB format
	availableDiskSize := float64(free) / humanize.GByte
	if availableDiskSize < constants.MinimumAvailableDiskSize {
		return fmt.Errorf(
			"there is not enough space to proceed with the installation, a minimum of %d GB is required to proceed",
			constants.MinimumAvailableDiskSize,
		)
	}
	progressPrinter.IncrementTracker("preflight-checks", 1)

	// Objects to check for
	// Repositories that will be created throughout the initialization process
	newRepositoryNames := []string{"gitops", "metaphor"}
	newTeamNames := []string{"admins", "developers"}

	// Check git credentials
	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider))
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.GitCredentialsCheckStarted, "")
		if len(cGitToken) == 0 {
			msg := fmt.Sprintf(
				"please set a %s_TOKEN environment variable to continue",
				strings.ToUpper(config.GitProvider),
			)
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
			return err
		}

		viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider), true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.GitCredentialsCheckCompleted, "")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg(fmt.Sprintf("already completed %s checks - continuing", config.GitProvider))
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}
	// Swap tokens for git protocol
	var gitopsRepoURL string
	executionControl = viper.GetBool("kubefirst-checks.kbot-setup")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.KbotSetupStarted, "")

		log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
		sshPrivateKey, sshPublicKey, err = utils.CreateSshKeyPair()
		if err != nil {
			telemetry.SendEvent(segClient, telemetry.KbotSetupFailed, err.Error())
			return err
		}
		log.Info().Msg("ssh key pair creation complete")

		viper.Set("kbot.private-key", sshPrivateKey)
		viper.Set("kbot.public-key", sshPublicKey)
		viper.Set("kbot.username", "kbot")
		viper.Set("kubefirst-checks.kbot-setup", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.KbotSetupCompleted, "")
		log.Info().Msg("kbot-setup complete")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already setup kbot user - continuing")
		progressPrinter.IncrementTracker("preflight-checks", 1)
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	telemetry.SendEvent(segClient, telemetry.InitCompleted, "")
	telemetry.SendEvent(segClient, telemetry.InitCompleted, "")

	// Swap tokens for git protocol
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
		ClusterId:                     clusterId,
		CloudProvider:                 k3d.CloudProvider,
	}

	if useTelemetryFlag {
		gitopsDirectoryTokens.UseTelemetry = "true"
	} else {
		gitopsDirectoryTokens.UseTelemetry = "false"
	}

	//* generate http credentials for git auth over https
	httpAuth := &githttps.BasicAuth{
		Username: cGitUser,
		Password: cGitToken,
	}

	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := k3d.DownloadTools(clusterNameFlag, config.GitProvider, cGitOwner, config.ToolsDir, config.GitProtocol)
		if err != nil {
			return err
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}
	progressPrinter.IncrementTracker("preflight-checks", 1)

	metaphorTemplateTokens := k3d.MetaphorTokenValues{
		ClusterName:                   clusterNameFlag,
		CloudRegion:                   cloudRegionFlag,
		ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", containerRegistryHost, cGitOwner),
		DomainName:                    k3d.DomainName,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", k3d.DomainName),
		MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", k3d.DomainName),
		MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", k3d.DomainName),
	}

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {
	progressPrinter.IncrementTracker("preflight-checks", 1)
	progressPrinter.IncrementTracker("preflight-checks", 1)
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
			config.DestinationGitopsRepoURL, // default to https for git interactions when creating remotes
			config.GitopsDir,
			gitopsTemplateBranchFlag,
			gitopsTemplateURLFlag,
			config.DestinationMetaphorRepoURL, // default to https for git interactions when creating remotes
			config.K1Dir,
			&gitopsDirectoryTokens,
			config.MetaphorDir,
			&metaphorTemplateTokens,
			gitProtocolFlag,
			removeAtlantis,
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

	progressPrinter.AddTracker("applying-git-terraform", fmt.Sprintf("Applying %s Terraform", config.GitProvider), 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	switch config.GitProvider {
	case "github":
		// //* create teams and repositories in github
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
		if !executionControl {
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitHub resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			// tfEnvs = k3d.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITHUB_TOKEN"] = cGitToken
			tfEnvs["GITHUB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["AWS_ACCESS_KEY_ID"] = constants.MinioDefaultUsername
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = constants.MinioDefaultPassword
			tfEnvs["TF_VAR_aws_access_key_id"] = constants.MinioDefaultUsername
			tfEnvs["TF_VAR_aws_secret_access_key"] = constants.MinioDefaultPassword
			// Erase public key to prevent it from being created if the git protocol argument is set to htps
			switch config.GitProtocol {
			case "https":
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
			}
			err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
				telemetry.SendEvent(segClient, telemetry.GitTerraformApplyFailed, msg)
				return errors.New(msg)
			}

			log.Info().Msgf("created git repositories for github.com/%s", cGitOwner)
			viper.Set("kubefirst-checks.terraform-apply-github", true)
			viper.WriteConfig()
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyCompleted, "")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		} else {
			log.Info().Msg("already created GitHub Terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	case "gitlab":
		// //* create teams and repositories in gitlab
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-gitlab")
		if !executionControl {
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyStarted, "")

			log.Info().Msg("Creating GitLab resources with Terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GITLAB_OWNER"] = gitlabGroupFlag
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(cGitlabOwnerGroupID)
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["AWS_ACCESS_KEY_ID"] = constants.MinioDefaultUsername
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = constants.MinioDefaultPassword
			tfEnvs["TF_VAR_aws_access_key_id"] = constants.MinioDefaultUsername
			tfEnvs["TF_VAR_aws_secret_access_key"] = constants.MinioDefaultPassword
			// Erase public key to prevent it from being created if the git protocol argument is set to htps
			switch config.GitProtocol {
			case "https":
				tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
			}
			err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
				telemetry.SendEvent(segClient, telemetry.GitTerraformApplyFailed, msg)
				return errors.New(msg)
			}

			log.Info().Msgf("created git projects and groups for gitlab.com/%s", gitlabGroupFlag)
			viper.Set("kubefirst-checks.terraform-apply-gitlab", true)
			viper.WriteConfig()
			telemetry.SendEvent(segClient, telemetry.GitTerraformApplyCompleted, "")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		} else {
			log.Info().Msg("already created GitLab Terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	}

	//* push detokenized gitops-template repository content to new remote
	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %s", config.DestinationGitopsRepoGitURL)
	log.Info().Msgf("referencing metaphor repository: %s", config.DestinationMetaphorRepoURL)

	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.GitopsRepoPushStarted, "")

		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		err = utils.EvalSSHKey(&types.EvalSSHKeyRequest{
			GitProvider:     gitProviderFlag,
			GitlabGroupFlag: gitlabGroupFlag,
			GitToken:        cGitToken,
		})
		if err != nil {
			return err
		}

		// Push to remotes and use https
		// Push gitops repo to remote
		err = gitopsRepo.Push(
			&git.PushOptions{
				RemoteName: config.GitProvider,
				Auth:       httpAuth,
			},
		)
		if err != nil {
			msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", config.DestinationGitopsRepoGitURL, err)
			telemetry.SendEvent(segClient, telemetry.GitopsRepoPushFailed, msg)
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
			msg := fmt.Sprintf("error pushing detokenized metaphor repository to remote %s: %s", config.DestinationMetaphorRepoURL, err)
			telemetry.SendEvent(segClient, telemetry.GitopsRepoPushFailed, msg)
			if !strings.Contains(msg, "already up-to-date") {
				log.Panic().Msg(msg)
			}
		}
		log.Info().Msgf("successfully pushed gitops and metaphor repositories to https://%s/%s", cGitHost, cGitOwner)

		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.GitopsRepoPushCompleted, "")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1) // todo verify this tracker didnt lose one
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)
	}

	//* create k3d resources

	progressPrinter.AddTracker("creating-k3d-cluster", "Creating k3d cluster", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		telemetry.SendEvent(segClient, telemetry.CloudTerraformApplyStarted, "")

		log.Info().Msg("Creating k3d cluster")

		err := k3d.ClusterCreate(clusterNameFlag, config.K1Dir, config.K3dClient, config.Kubeconfig)
		if err != nil {
			msg := fmt.Sprintf("error creating k3d resources with k3d client %s: %s", config.K3dClient, err)
			viper.Set("kubefirst-checks.create-k3d-cluster-failed", true)
			viper.WriteConfig()
			telemetry.SendEvent(segClient, telemetry.CloudTerraformApplyFailed, msg)
			return errors.New(msg)
		}

		log.Info().Msg("successfully created k3d cluster")
		viper.Set("kubefirst-checks.create-k3d-cluster", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.CloudTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("creating-k3d-cluster", 1)
	} else {
		log.Info().Msg("already created k3d cluster resources")
		progressPrinter.IncrementTracker("creating-k3d-cluster", 1)
	}

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	// kubernetes.BootstrapSecrets
	progressPrinter.AddTracker("bootstrapping-kubernetes-resources", "Bootstrapping Kubernetes resources", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := k3d.GenerateTLSSecrets(kcfg.Clientset, *config)
		if err != nil {
			return err
		}

		err = k3d.AddK3DSecrets(
			atlantisWebhookSecret,
			viper.GetString("kbot.public-key"),
			gitopsRepoURL,
			viper.GetString("kbot.private-key"),
			config.GitProvider,
			cGitUser,
			cGitOwner,
			config.Kubeconfig,
			cGitToken,
		)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	} else {
		log.Info().Msg("already added secrets to k3d cluster")
		progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
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
	// 	ssl.Restore(config.SSLBackupDir, k3d.DomainName, config.Kubeconfig)
	// } else {
	// 	log.Info().Msg("no files found in secrets directory, continuing")
	// }

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

	// k3d Readiness checks
	progressPrinter.AddTracker("verifying-k3d-cluster-readiness", "Verifying Kubernetes cluster is ready", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// traefik
	traefikDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"traefik",
		"kube-system",
		240,
	)
	if err != nil {
		log.Error().Msgf("error finding traefik deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, traefikDeployment, 240)
	if err != nil {
		log.Error().Msgf("error waiting for traefik deployment ready state: %s", err)
		return err
	}
	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness", 1)

	// metrics-server
	metricsServerDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"k8s-app",
		"metrics-server",
		"kube-system",
		240,
	)
	if err != nil {
		log.Error().Msgf("error finding metrics-server deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, metricsServerDeployment, 240)
	if err != nil {
		log.Error().Msgf("error waiting for metrics-server deployment ready state: %s", err)
		return err
	}
	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness", 1)

	time.Sleep(time.Second * 20)

	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness", 1)

	progressPrinter.AddTracker("installing-argo-cd", "Installing and configuring Argo CD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	argoCDInstallPath := fmt.Sprintf("github.com:konstructio/manifests/argocd/k3d?ref=%s", constants.KubefirstManifestRepoRef)
	//* install argo
	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.ArgoCDInstallStarted, "")

		log.Info().Msgf("installing argocd")

		// Build and apply manifests
		yamlData, err := kcfg.KustomizeBuild(argoCDInstallPath)
		if err != nil {
			return err
		}
		output, err := kcfg.SplitYAMLFile(yamlData)
		if err != nil {
			return err
		}
		err = kcfg.ApplyObjects("", output)
		if err != nil {
			telemetry.SendEvent(segClient, telemetry.ArgoCDInstallFailed, err.Error())
			return err
		}

		viper.Set("kubefirst-checks.argocd-install", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.ArgoCDInstallCompleted, "")
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

	var argocdPassword string
	//* argocd pods are ready, get and set credentials
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

		// Test https to argocd
		var argoCDToken string
		// only the host, not the protocol
		err := utils.TestEndpointTLS(strings.Replace(k3d.ArgocdURL, "https://", "", 1))
		if err != nil {
			argoCDStopChannel := make(chan struct{}, 1)
			log.Info().Msgf("argocd not available via https, using http")
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
			argoCDToken, err = argocd.GetArgocdTokenV2(httpClient, argoCDHTTPURL, "admin", argocdPassword)
			if err != nil {
				return err
			}
		} else {
			argoCDToken, err = argocd.GetArgocdTokenV2(httpClient, k3d.ArgocdURL, "admin", argocdPassword)
			if err != nil {
				return err
			}
		}

		log.Info().Msg("argocd admin auth token set")

		viper.Set("components.argocd.auth-token", argoCDToken)
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

		if os.Getenv("SKIP_ARGOCD_LAUNCH") != "true" || !ciFlag {
			err = utils.OpenBrowser(constants.ArgoCDLocalURLTLS)
			if err != nil {
				log.Error().Err(err).Msg("")
			}
		}
	}

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		telemetry.SendEvent(segClient, telemetry.CreateRegistryStarted, "")
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return err
		}

		log.Info().Msg("applying the registry application to argocd")
		registryApplicationObject := argocd.GetArgoCDApplicationObject(gitopsRepoURL, fmt.Sprintf("registry/%s", clusterNameFlag))

		err = k3d.RestartDeployment(context.Background(), kcfg.Clientset, "argocd", "argocd-applicationset-controller")
		if err != nil {
			return fmt.Errorf("error in restarting argocd controller %w", err)
		}

		err = wait.PollImmediate(5*time.Second, 20*time.Second, func() (bool, error) {
			_, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
			if err != nil {
				if errors.Is(err, syscall.ECONNREFUSED) {
					return false, nil // retry if we can't connect to it
				}

				if apierrors.IsAlreadyExists(err) {
					return true, nil // application already exists
				}

				return false, fmt.Errorf("error creating argocd application : %w", err)
			}
			return true, nil
		})
		if err != nil {
			return fmt.Errorf("error creating argocd application : %w", err)
		}

		log.Info().Msg("Argo CD application created successfully")
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.CreateRegistryCompleted, "")
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
		120,
	)
	if err != nil {
		log.Error().Msgf("Error finding Vault StatefulSet: %s", err)
		return err
	}
	_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 120, true)
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
		telemetry.SendEvent(segClient, telemetry.VaultInitializationStarted, "")

		// Initialize and unseal Vault
		vaultHandlerPath := "github.com:kubefirst/manifests.git/vault-handler/replicas-1"

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
			telemetry.SendEvent(segClient, telemetry.VaultInitializationFailed, msg)
			log.Fatal().Msg(msg)
		}

		viper.Set("kubefirst-checks.vault-initialized", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.VaultInitializationCompleted, "")
		progressPrinter.IncrementTracker("configuring-vault", 1)
	} else {
		log.Info().Msg("vault is already initialized - skipping")
		progressPrinter.IncrementTracker("configuring-vault", 1)
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

	// Initialize minio client object.
	minioClient, err := minio.New(constants.MinioPortForwardEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(constants.MinioDefaultUsername, constants.MinioDefaultPassword, ""),
		Secure: false,
		Region: constants.MinioRegion,
	})
	if err != nil {
		log.Info().Msgf("Error creating Minio client: %s", err)
	}

	// define upload object
	objectName := fmt.Sprintf("terraform/%s/terraform.tfstate", config.GitProvider)
	filePath := config.K1Dir + fmt.Sprintf("/gitops/%s", objectName)
	contentType := "xl.meta"
	bucketName := "kubefirst-state-store"
	log.Info().Msgf("BucketName: %s", bucketName)

	viper.Set("kubefirst.state-store.name", bucketName)
	viper.Set("kubefirst.state-store.hostname", "minio-console.kubefirst.dev")
	viper.Set("kubefirst.state-store-creds.access-key-id", constants.MinioDefaultUsername)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", constants.MinioDefaultPassword)

	// Upload the zip file with FPutObject
	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		log.Info().Msgf("Error uploading to Minio bucket: %s", err)
	}

	log.Printf("Successfully uploaded %s to bucket %s\n", objectName, info.Bucket)

	progressPrinter.IncrementTracker("configuring-vault", 1)

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

	// Retrieve root token from init step
	var vaultRootToken string
	secData, err := k8s.ReadSecretV2(kcfg.Clientset, "vault", "vault-unseal-secret")
	if err != nil {
		return err
	}

	vaultRootToken = secData["root-token"]

	// Parse k3d api endpoint from kubeconfig
	// In this case, we need to get the IP of the in-cluster API server to provide to Vault
	// to work with Kubernetes auth
	kubernetesInClusterAPIService, err := k8s.ReadService(config.Kubeconfig, "default", "kubernetes")
	if err != nil {
		log.Error().Msgf("error looking up kubernetes api server service: %s", err)
		return err
	}

	err = utils.TestEndpointTLS(strings.Replace(k3d.VaultURL, "https://", "", 1))
	if err != nil {
		return fmt.Errorf(
			"unable to reach vault over https - this is likely due to the mkcert certificate store missing. please install it via `%s -install`", config.MkCertClient,
		)
	}

	//* configure vault with terraform
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
		// tfEnvs["TF_LOG"] = "DEBUG"

		tfEntrypoint := config.GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			telemetry.SendEvent(segClient, telemetry.VaultTerraformApplyStarted, err.Error())
			return err
		}

		log.Info().Msg("vault terraform executed successfully")
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.VaultTerraformApplyCompleted, "")
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
			return err
		}
		log.Info().Msg("executed users terraform successfully")
		// progressPrinter.IncrementTracker("step-users", 1)
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
		telemetry.SendEvent(segClient, telemetry.UsersTerraformApplyCompleted, "")
		progressPrinter.IncrementTracker("creating-users", 1)
	} else {
		log.Info().Msg("already created users with terraform")
		progressPrinter.IncrementTracker("creating-users", 1)
	}

	// PostRun string replacement
	progressPrinter.AddTracker("wrapping-up", "Wrapping up", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	err = k3d.PostRunPrepareGitopsRepository(clusterNameFlag,
		config.GitopsDir,
		&gitopsDirectoryTokens,
	)
	if err != nil {
		log.Info().Msgf("Error detokenize post run: %s", err)
	}
	gitopsRepo, err := git.PlainOpen(config.GitopsDir)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
	}
	// check if file exists before rename
	_, err = os.Stat(fmt.Sprintf("%s/terraform/%s/remote-backend.md", config.GitopsDir, config.GitProvider))
	if err == nil {
		err = os.Rename(fmt.Sprintf("%s/terraform/%s/remote-backend.md", config.GitopsDir, config.GitProvider), fmt.Sprintf("%s/terraform/%s/remote-backend.tf", config.GitopsDir, config.GitProvider))
		if err != nil {
			return err
		}
	}
	viper.Set("kubefirst-checks.post-detokenize", true)
	viper.WriteConfig()

	// Final gitops repo commit and push
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content post run")
	if err != nil {
		return err
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: config.GitProvider,
		Auth:       httpAuth,
	})
	if err != nil {
		log.Info().Msgf("Error pushing repo: %s", err)
	}

	progressPrinter.IncrementTracker("wrapping-up", 1)

	// Wait for console Deployment Pods to transition to Running
	argoDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"argo",
		"argo",
		1200,
	)
	if err != nil {
		log.Error().Msgf("Error finding argo workflows Deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, argoDeployment, 120)
	if err != nil {
		log.Error().Msgf("Error waiting for argo workflows Deployment ready state: %s", err)
		return err
	}

	// Set flags used to track status of active options
	utils.SetClusterStatusFlags(k3d.CloudProvider, config.GitProvider)

	cluster := utilities.CreateClusterRecordFromRaw(useTelemetryFlag, cGitOwner, cGitUser, cGitToken, cGitlabOwnerGroupID, gitopsTemplateURLFlag, gitopsTemplateBranchFlag, catalogApps)

	err = utilities.ExportCluster(cluster, kcfg)
	if err != nil {
		log.Error().Err(err).Msg("error exporting cluster object")
		viper.Set("kubefirst.setup-complete", false)
		viper.Set("kubefirst-checks.cluster-install-complete", false)
		viper.WriteConfig()
		return err
	} else {
		kubefirstDeployment, err := k8s.ReturnDeploymentObject(
			kcfg.Clientset,
			"app.kubernetes.io/instance",
			"kubefirst",
			"kubefirst",
			600,
		)
		if err != nil {
			log.Error().Msgf("Error finding kubefirst Deployment: %s", err)
			return err
		}
		_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, kubefirstDeployment, 120)
		if err != nil {
			log.Error().Msgf("Error waiting for kubefirst Deployment ready state: %s", err)
			return err
		}
		progressPrinter.IncrementTracker("wrapping-up", 1)

		err = utils.OpenBrowser(constants.KubefirstConsoleLocalURLTLS)
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		// Mark cluster install as complete
		telemetry.SendEvent(segClient, telemetry.ClusterInstallCompleted, "")
		viper.Set("kubefirst-checks.cluster-install-complete", true)
		viper.WriteConfig()

		log.Info().Msg("kubefirst installation complete")
		log.Info().Msg("welcome to your new kubefirst platform running in K3d")
		time.Sleep(time.Second * 1) // allows progress bars to finish

		reports.LocalHandoffScreenV2(viper.GetString("components.argocd.password"), clusterNameFlag, gitDestDescriptor, cGitOwner, config, ciFlag)

		if ciFlag {
			progress.Progress.Quit()
		}
	}

	return nil
}