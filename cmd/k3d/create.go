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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/docker"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/github"
	gitlab "github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/segment"
	"github.com/kubefirst/kubefirst/internal/services"
	internalssh "github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cancelContext context.CancelFunc
)

func runK3d(cmd *cobra.Command, args []string) error {
	helpers.DisplayLogHints()

	clusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		return err
	}

	clusterTypeFlag, err := cmd.Flags().GetString("cluster-type")
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

	// Either user or org can be specified for github, not both
	if githubOrgFlag != "" && githubUserFlag != "" {
		return errors.New("only one of --github-user or --github-org can be supplied")
	}

	// Check for existing port forwards before continuing
	err = k8s.CheckForExistingPortForwards(8080, 8200, 9094)
	if err != nil {
		return fmt.Errorf("%s - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing", err.Error())
	}

	// Verify Docker is running
	dcli := docker.DockerClientWrapper{
		Client: docker.NewDockerClient(),
	}
	_, err = dcli.CheckDockerReady()
	if err != nil {
		return err
	}

	// Global context
	var ctx context.Context
	ctx, cancelContext = context.WithCancel(context.Background())

	// Clients
	httpClient := http.DefaultClient
	segmentClient := &segment.Client
	var segmentMsg string

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	// Store flags for application state maintenance
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", k3d.DomainName)
	viper.Set("flags.dry-run", dryRunFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
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

		log.Info().Msgf("ignoring %s", cGitOwner)
	case "gitlab":
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
		log.Info().Msgf("set gitlab owner to %s", cGitOwner)

		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		cGitUser = user.Username
		containerRegistryHost = "registry.gitlab.com"
		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
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
	config := k3d.GetConfig(gitProviderFlag, cGitOwner)

	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}

	if useTelemetryFlag {
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, clusterId, clusterTypeFlag, k3d.DomainName, gitProviderFlag, kubefirstTeam, pkg.MetricInitStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	}

	// Progress output
	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 5)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	progressPrinter.IncrementTracker("preflight-checks", 1)

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

	atlantisWebhookSecret := viper.GetString("secrets.atlantis-webhook")
	if atlantisWebhookSecret == "" {
		atlantisWebhookSecret = pkg.Random(20)
		viper.Set("secrets.atlantis-webhook", atlantisWebhookSecret)
		viper.WriteConfig()
	}

	log.Info().Msg("checking authentication to required providers")

	// check disk
	free, err := pkg.GetAvailableDiskSize()
	if err != nil {
		return err
	}

	// convert available disk size to GB format
	availableDiskSize := float64(free) / humanize.GByte
	if availableDiskSize < pkg.MinimumAvailableDiskSize {
		return fmt.Errorf(
			"there is not enough space to proceed with the installation, a minimum of %d GB is required to proceed",
			pkg.MinimumAvailableDiskSize,
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
				responseStatusCode := githubSession.CheckRepoExists(cGitOwner, repositoryName)

				// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository
				repositoryExistsStatusCode := 200
				repositoryDoesNotExistStatusCode := 404

				if responseStatusCode == repositoryExistsStatusCode {
					log.Info().Msgf("repository https://github.com/%s/%s exists", cGitOwner, repositoryName)
					errorMsg = errorMsg + fmt.Sprintf("https://github.com/%s/%s\n\t", cGitOwner, repositoryName)
					newRepositoryExists = true
				} else if responseStatusCode == repositoryDoesNotExistStatusCode {
					log.Info().Msgf("repository https://github.com/%s/%s does not exist, continuing", cGitOwner, repositoryName)
				}
			}
			if newRepositoryExists {
				return errors.New(errorMsg)
			}

			newTeamExists := false
			errorMsg = "the following teams must be removed before continuing with your kubefirst installation.\n\t"

			for _, teamName := range newTeamNames {
				responseStatusCode := githubSession.CheckTeamExists(cGitOwner, teamName)

				// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#get-a-team-by-name
				teamExistsStatusCode := 200
				teamDoesNotExistStatusCode := 404

				if responseStatusCode == teamExistsStatusCode {
					log.Info().Msgf("team https://github.com/%s/%s exists", cGitOwner, teamName)
					errorMsg = errorMsg + fmt.Sprintf("https://github.com/orgs/%s/teams/%s\n\t", cGitOwner, teamName)
					newTeamExists = true
				} else if responseStatusCode == teamDoesNotExistStatusCode {
					log.Info().Msgf("https://github.com/orgs/%s/teams/%s does not exist, continuing", cGitOwner, teamName)
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

	// todo this is actually your personal account
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

	gitopsTemplateTokens := k3d.GitopsTokenValues{
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
		GitopsRepoGitURL:              config.DestinationGitopsRepoGitURL,
		GitProvider:                   config.GitProvider,
		ClusterId:                     clusterId,
		CloudProvider:                 k3d.CloudProvider,
	}

	if useTelemetryFlag {
		gitopsTemplateTokens.UseTelemetry = "true"

		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, clusterId, clusterTypeFlag, k3d.DomainName, gitProviderFlag, kubefirstTeam, pkg.MetricInitCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, clusterId, clusterTypeFlag, k3d.DomainName, gitProviderFlag, kubefirstTeam, pkg.MetricMgmtClusterInstallStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	} else {
		gitopsTemplateTokens.UseTelemetry = "false"
	}

	//* generate public keys for ssh
	publicKeys, err := gitssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := k3d.DownloadTools(config.GitProvider, cGitOwner, config.ToolsDir)
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
	progressPrinter.AddTracker("cloning-and-formatting-git-repositories", "Cloning and formatting git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {
		log.Info().Msg("generating your new gitops repository")
		err := k3d.PrepareGitRepositories(
			config.GitProvider,
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
			// tfEnvs = k3d.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITHUB_TOKEN"] = cGitToken
			tfEnvs["GITHUB_OWNER"] = cGitOwner
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
			tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
			tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
			err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
			if err != nil {
				return fmt.Errorf("error creating github resources with terraform %s: %s", tfEntrypoint, err)
			}

			log.Info().Msgf("created git repositories for github.com/%s", cGitOwner)
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
			tfEnvs["GITLAB_TOKEN"] = cGitToken
			tfEnvs["GITLAB_OWNER"] = gitlabGroupFlag
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(cGitlabOwnerGroupID)
			tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
			tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
			tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
			err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
			if err != nil {
				return fmt.Errorf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err)
			}

			log.Info().Msgf("created git projects and groups for gitlab.com/%s", gitlabGroupFlag)
			viper.Set("kubefirst-checks.terraform-apply-gitlab", true)
			viper.WriteConfig()
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		} else {
			log.Info().Msg("already created gitlab terraform resources")
			progressPrinter.IncrementTracker("applying-git-terraform", 1)
		}
	}

	//* push detokenized gitops-template repository content to new remote
	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %s", config.DestinationGitopsRepoGitURL)
	log.Info().Msgf("referencing metaphor repository: %s", config.DestinationMetaphorRepoGitURL)

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

			var keyName = "kubefirst-k3d-ssh-key"
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
				viper.Set("kbot.gitlab-user-based-ssh-key-title", "kubefirst-k3d-ssh-key")
				viper.WriteConfig()
			}
		}

		// Push gitops repo to remote
		err = gitopsRepo.Push(
			&git.PushOptions{
				RemoteName: config.GitProvider,
				Auth:       publicKeys,
			},
		)
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s: %s", config.DestinationGitopsRepoGitURL, err)
		}

		// push metaphor repo to remote
		err = metaphorRepo.Push(
			&git.PushOptions{
				RemoteName: "origin",
				Auth:       publicKeys,
			},
		)
		if err != nil {
			log.Panic().Msgf("error pushing detokenized metaphor repository to remote %s: %s", config.DestinationMetaphorRepoGitURL, err)
		}

		log.Info().Msgf("successfully pushed gitops and metaphor repositories to git@%s/%s", cGitHost, cGitOwner)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1) // todo verify this tracker didnt lose one
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
		progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)
	}

	//* create k3d resources

	progressPrinter.AddTracker("creating-k3d-cluster", "Creating K3d cluster", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.terraform-apply-k3d") {
		log.Info().Msg("Creating k3d cluster")

		err := k3d.ClusterCreate(clusterNameFlag, config.K1Dir, config.K3dClient, config.Kubeconfig)
		if err != nil {
			viper.Set("kubefirst-checks.terraform-apply-k3d-failed", true)
			viper.WriteConfig()

			return err
		}

		log.Info().Msg("successfully created k3d cluster")
		viper.Set("kubefirst-checks.terraform-apply-k3d", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("creating-k3d-cluster", 1)
	} else {
		log.Info().Msg("already created k3d cluster resources")
		progressPrinter.IncrementTracker("creating-k3d-cluster", 1)
	}

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
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
			config.DestinationGitopsRepoGitURL,
			viper.GetString("kbot.private-key"),
			false,
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

	// GitLab Deploy Tokens
	// Handle secret creation for buildkit
	createTokensFor := []string{"metaphor"}
	switch config.GitProvider {
	// GitHub docker auth secret
	// Buildkit requires a specific format for Docker auth created as a secret
	// For GitHub, this becomes the provided token (pat)
	case "github":
		usernamePasswordString := fmt.Sprintf("%s:%s", cGitUser, cGitToken)
		usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"username": "%s", "password": "%s", "email": "%s", "auth": "%s"}}}`, containerRegistryHost, cGitUser, cGitToken, "k-bot@example.com", usernamePasswordStringB64)

		for _, repository := range createTokensFor {
			// Create argo workflows pull secret
			// This is formatted to work with buildkit
			argoDeployTokenSecret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-deploy", repository), Namespace: "argo"},
				Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
				Type:       "Opaque",
			}
			err = k8s.CreateSecretV2(kcfg.Clientset, argoDeployTokenSecret)
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
					err = k8s.CreateSecretV2(kcfg.Clientset, deployTokenSecret)
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
				err = k8s.CreateSecretV2(kcfg.Clientset, argoDeployTokenSecret)
				if err != nil {
					log.Error().Msgf("error while creating secret for project deploy token: %s", err)
				}
			}
		}
	}
	progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)

	progressPrinter.AddTracker("installing-argo-cd", "Installing and configuring ArgoCD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	argoCDInstallPath := "github.com:kubefirst/manifests/argocd/k3d?ref=argocd"

	//* install argocd
	executionControl = viper.GetBool("kubefirst-checks.argocd-install")
	if !executionControl {
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
	_, err = k8s.VerifyArgoCDReadiness(kcfg.Clientset, true)
	if err != nil {
		log.Error().Msgf("error waiting for ArgoCD to become ready: %s", err)
		return err
	}

	if configs.K1Version == "development" {
		err = pkg.OpenBrowser(pkg.ArgoCDLocalURLTLS)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
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
		err := helpers.TestEndpointTLS(strings.Replace(k3d.ArgocdURL, "https://", "", 1))
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

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return err
		}

		log.Info().Msg("applying the registry application to argocd")
		registryApplicationObject := argocd.GetArgoCDApplicationObject(config.DestinationGitopsRepoGitURL, fmt.Sprintf("registry/%s", clusterNameFlag))
		_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
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
			log.Fatal().Msgf("could not run vault unseal job: %s", err)
		}

		viper.Set("kubefirst-checks.vault-initialized", true)
		viper.WriteConfig()
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
	minioClient, err := minio.New(pkg.MinioPortForwardEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(pkg.MinioDefaultUsername, pkg.MinioDefaultPassword, ""),
		Secure: false,
		Region: pkg.MinioRegion,
	})

	if err != nil {
		log.Info().Msgf("Error creating Minio client: %s", err)
	}

	//define upload object
	objectName := fmt.Sprintf("terraform/%s/terraform.tfstate", config.GitProvider)
	filePath := config.K1Dir + fmt.Sprintf("/gitops/%s", objectName)
	contentType := "xl.meta"
	bucketName := "kubefirst-state-store"
	log.Info().Msgf("BucketName: %s", bucketName)

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
		log.Error().Msgf("error looking up kubernetes api server service: %s")
		return err
	}

	//* configure vault with terraform
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

		//* run vault terraform
		log.Info().Msg("configuring vault with terraform")

		usernamePasswordString := fmt.Sprintf("%s:%s", cGitUser, cGitToken)
		base64DockerAuth := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

		tfEnvs := map[string]string{}
		//tfEnvs = k3d.GetVaultTerraformEnvs(config, tfEnvs)
		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
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
		tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
		tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
		tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
		tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
		// tfEnvs["TF_LOG"] = "DEBUG"

		if config.GitProvider == "gitlab" {
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(cGitlabOwnerGroupID)
		}

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
		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
		tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = vaultRootToken
		tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = vaultRootToken
		tfEnvs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = cGitToken
		tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = cGitOwner

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

	//PostRun string replacement
	progressPrinter.AddTracker("wrapping-up", "Wrapping up", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	err = k3d.PostRunPrepareGitopsRepository(clusterNameFlag,
		config.GitopsDir,
		&gitopsTemplateTokens,
	)
	if err != nil {
		log.Info().Msgf("Error detokenize post run: %s", err)
	}
	gitopsRepo, err := git.PlainOpen(config.GitopsDir)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
	}
	//check if file exists before rename
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
		Auth:       publicKeys,
	})
	if err != nil {
		log.Info().Msgf("Error pushing repo: %s", err)
	}

	progressPrinter.IncrementTracker("wrapping-up", 1)

	// Wait for console Deployment Pods to transition to Running
	consoleDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		600,
	)
	if err != nil {
		log.Error().Msgf("Error finding console Deployment: %s", err)
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, consoleDeployment, 120)
	if err != nil {
		log.Error().Msgf("Error waiting for console Deployment ready state: %s", err)
		return err
	}

	//* console port-forward
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

	progressPrinter.IncrementTracker("wrapping-up", 1)

	log.Info().Msg("kubefirst installation complete")
	log.Info().Msg("welcome to your new kubefirst platform running in K3d")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	if useTelemetryFlag {
		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, clusterId, clusterTypeFlag, k3d.DomainName, gitProviderFlag, kubefirstTeam, pkg.MetricMgmtClusterInstallCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
	}

	// Set flags used to track status of active options
	helpers.SetCompletionFlags(k3d.CloudProvider, config.GitProvider)

	reports.LocalHandoffScreenV2(viper.GetString("components.argocd.password"), clusterNameFlag, gitDestDescriptor, cGitOwner, config, dryRunFlag, false)

	time.Sleep(time.Millisecond * 100) // allows progress bars to finish

	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Info().Msgf("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return nil
}
