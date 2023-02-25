package k3d

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"

	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	gitlab "github.com/kubefirst/kubefirst/internal/gitlabcloud"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/services"
	internalssh "github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cancelContext context.CancelFunc
)

func runK3d(cmd *cobra.Command, args []string) error {
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

	githubOwnerFlag, err := cmd.Flags().GetString("github-owner")
	if err != nil {
		return err
	}

	gitlabOwnerFlag, err := cmd.Flags().GetString("gitlab-owner")
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

	// Set git handlers
	switch gitProviderFlag {
	case "github":
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		// get github data to set user based on the provided token
		log.Info().Msg("verifying github authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			return err
		}
		// today we override the owner to be the user's token by default
		githubOwnerFlag = githubUser
		viper.Set("flags.github-owner", githubOwnerFlag)
	case "gitlab":
		viper.Set("flags.gitlab-owner", gitlabOwnerFlag)
	}

	// required for destroy command
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", k3d.DomainName)
	viper.Set("flags.dry-run", dryRunFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.WriteConfig()

	// creates a new context, and a cancel function that allows canceling the context. The context is passed as an
	// argument to the RunNgrok function, which is then started in a new goroutine.
	var ctx context.Context
	ctx, cancelContext = context.WithCancel(context.Background())
	go pkg.RunNgrok(ctx)
	if err != nil {
		return err
	}
	ngrokHost := viper.GetString("ngrok.host")

	// Switch based on git provider, set params
	var cGitHost, cGitOwner, cGitUser, cGitToken string
	switch gitProviderFlag {
	case "github":
		cGitHost = k3d.GithubHost
		cGitOwner = githubOwnerFlag
		cGitUser = githubOwnerFlag
		cGitToken = os.Getenv("GITHUB_TOKEN")
	case "gitlab":
		cGitHost = k3d.GitlabHost
		cGitOwner = gitlabOwnerFlag
		cGitUser = gitlabOwnerFlag
		cGitToken = os.Getenv("GITLAB_TOKEN")
	default:
		log.Error().Msgf("invalid git provider option")
	}

	// Instantiate K3d config
	config := k3d.GetConfig(gitProviderFlag, cGitOwner)
	gitopsTemplateTokens := k3d.GitopsTokenValues{}
	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(k3d.DomainName, pkg.MetricInitStarted, k3d.CloudProvider, config.GitProvider); err != nil {
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

	// Check git credentials
	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider))
	if !executionControl {
		if len(cGitToken) == 0 {
			return errors.New(
				fmt.Sprintf(
					"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init",
					strings.ToUpper(config.GitProvider),
				),
			)
		}

		// Objects to check for
		newRepositoryNames := []string{"gitops", "metaphor-frontend"}
		newTeamNames := []string{"admins", "developers"}

		switch config.GitProvider {
		case "github":
			githubWrapper := githubWrapper.New()
			newRepositoryExists := false
			// todo hoist to globals
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

			newTeamExists := false
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
		case "gitlab":
			gl := gitlab.GitLabWrapper{
				Client: gitlab.NewGitLabClient(cGitToken),
			}

			// Check for existing base projects
			projects, err := gl.GetProjects()
			if err != nil {
				log.Fatal().Msgf("couldn't get gitlab projects: %s", err)
			}
			for _, repositoryName := range newRepositoryNames {
				found, err := gl.FindProjectInGroup(projects, repositoryName)
				if err != nil {
					log.Info().Msg(err.Error())
				}
				if found {
					return errors.New(fmt.Sprintf("project %s already exists and will need to be deleted before continuing", repositoryName))
				}
			}

			// Check for existing base projects
			allgroups, err := gl.GetGroups()
			if err != nil {
				log.Fatal().Msgf("could not read gitlab groups: %s", err)
			}
			gid, err := gl.GetGroupID(allgroups, gitlabOwnerFlag)
			if err != nil {
				log.Fatal().Msgf("could not get group id for primary group: %s", err)
			}
			subgroups, err := gl.GetSubGroups(gid)
			if err != nil {
				log.Fatal().Msgf("couldn't get gitlab projects: %s", err)
			}
			for _, teamName := range newRepositoryNames {
				for _, sg := range subgroups {
					if sg.Name == teamName {
						return errors.New(fmt.Sprintf("subgroup %s already exists and will need to be deleted before continuing", teamName))
					}
				}
			}
		}

		viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider), true)
		viper.WriteConfig()
	} else {
		log.Info().Msg(fmt.Sprintf("already completed %s checks - continuing", config.GitProvider))
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
	} else {
		log.Info().Msg("already setup kbot user - continuing")
	}
	log.Info().Msg("validation and kubefirst cli environment check is complete")

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(k3d.DomainName, pkg.MetricInitCompleted, k3d.CloudProvider, config.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(k3d.DomainName, pkg.MetricMgmtClusterInstallStarted, k3d.CloudProvider, config.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	//* generate public keys for ssh
	publicKeys, err := gitssh.NewPublicKeys("git", []byte(viper.GetString("kbot.private-key")), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(k3d.DomainName, pkg.MetricMgmtClusterInstallStarted, k3d.CloudProvider, config.GitProvider); err != nil {
			log.Info().Msg(err.Error())
		}
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

	// not sure if there is a better way to do this
	gitopsTemplateTokens.GithubOwner = githubOwnerFlag
	gitopsTemplateTokens.GithubUser = cGitUser
	gitopsTemplateTokens.GitlabOwner = gitlabOwnerFlag
	gitopsTemplateTokens.GitlabUser = cGitUser
	gitopsTemplateTokens.GitopsRepoGitURL = config.DestinationGitopsRepoGitURL
	gitopsTemplateTokens.DomainName = k3d.DomainName
	gitopsTemplateTokens.AtlantisAllowList = fmt.Sprintf("%s/%s/*", cGitHost, cGitOwner)
	gitopsTemplateTokens.NgrokHost = ngrokHost
	gitopsTemplateTokens.AlertsEmail = "REMOVE_THIS_VALUE"
	gitopsTemplateTokens.ClusterName = clusterNameFlag
	gitopsTemplateTokens.ClusterType = clusterTypeFlag
	gitopsTemplateTokens.GithubHost = k3d.GithubHost
	gitopsTemplateTokens.GitlabHost = k3d.GitlabHost
	gitopsTemplateTokens.ArgoWorkflowsIngressURL = fmt.Sprintf("https://argo.%s", k3d.DomainName)
	gitopsTemplateTokens.VaultIngressURL = fmt.Sprintf("https://vault.%s", k3d.DomainName)
	gitopsTemplateTokens.ArgocdIngressURL = fmt.Sprintf("https://argocd.%s", k3d.DomainName)
	gitopsTemplateTokens.AtlantisIngressURL = fmt.Sprintf("https://atlantis.%s", k3d.DomainName)
	gitopsTemplateTokens.MetaphorDevelopmentIngressURL = fmt.Sprintf("metaphor-development.%s", k3d.DomainName)
	gitopsTemplateTokens.MetaphorStagingIngressURL = fmt.Sprintf("metaphor-staging.%s", k3d.DomainName)
	gitopsTemplateTokens.MetaphorProductionIngressURL = fmt.Sprintf("metaphor-production.%s", k3d.DomainName)
	gitopsTemplateTokens.KubefirstVersion = configs.K1Version

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {
		log.Info().Msg("generating your new gitops repository")

		err := k3d.PrepareGitopsRepository(
			config.GitProvider,
			clusterNameFlag,
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

	atlantisWebhookURL := fmt.Sprintf("%s/events", viper.GetString("ngrok.host"))

	switch config.GitProvider {
	case "github":
		// //* create teams and repositories in github
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
		if !executionControl {
			log.Info().Msg("Creating github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			// tfEnvs = k3d.GetGithubTerraformEnvs(tfEnvs)
			tfEnvs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
			tfEnvs["GITHUB_OWNER"] = githubOwnerFlag
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
			tfEnvs["AWS_ACCESS_KEY_ID"] = "kray"
			tfEnvs["AWS_SECRET_ACCESS_KEY"] = "feedkraystars"
			tfEnvs["TF_VAR_aws_access_key_id"] = "kray"
			tfEnvs["TF_VAR_aws_secret_access_key"] = "feedkraystars"
			err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
			if err != nil {
				return errors.New(fmt.Sprintf("error creating github resources with terraform %s: %s", tfEntrypoint, err))
			}

			log.Info().Msgf("created git repositories and teams for github.com/%s", githubOwnerFlag)
			viper.Set("kubefirst-checks.terraform-apply-github", true)
			viper.WriteConfig()
		} else {
			log.Info().Msg("already created github terraform resources")
		}
	case "gitlab":
		// //* create teams and repositories in gitlab
		gl := gitlab.GitLabWrapper{
			Client: gitlab.NewGitLabClient(cGitToken),
		}
		allgroups, err := gl.GetGroups()
		if err != nil {
			log.Fatal().Msgf("could not read gitlab groups: %s", err)
		}
		gid, err := gl.GetGroupID(allgroups, gitlabOwnerFlag)
		if err != nil {
			log.Fatal().Msgf("could not get group id for primary group: %s", err)
		}
		executionControl = viper.GetBool("kubefirst-checks.terraform-apply-gitlab")
		if !executionControl {
			log.Info().Msg("Creating gitlab resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs["GITLAB_TOKEN"] = os.Getenv("GITLAB_TOKEN")
			tfEnvs["GITLAB_OWNER"] = gitlabOwnerFlag
			tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
			tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)
			err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
			if err != nil {
				return errors.New(fmt.Sprintf("error creating gitlab resources with terraform %s: %s", tfEntrypoint, err))
			}

			log.Info().Msgf("created git projects and groups for gitlab.com/%s", gitlabOwnerFlag)
			viper.Set("kubefirst-checks.terraform-apply-gitlab", true)
			viper.WriteConfig()
		} else {
			log.Info().Msg("already created gitlab terraform resources")
		}
	}

	//* push detokenized gitops-template repository content to new remote
	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}

		// For GitLab, we currently need to add an ssh key to the authenticating user
		if config.GitProvider == "gitlab" {
			gl := gitlab.GitLabWrapper{
				Client: gitlab.NewGitLabClient(cGitToken),
			}
			keys, err := gl.GetUserSSHKeys()
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
				err := gl.AddUserSSHKey(keyName, viper.GetString("kbot.public-key"))
				if err != nil {
					log.Fatal().Msgf("error adding ssh key %s: %s", keyName, err.Error())
				}
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

		log.Info().Msgf("successfully pushed gitops to git@g%s/%s/gitops", cGitHost, cGitOwner)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
	}

	metaphorTemplateTokens := k3d.MetaphorTokenValues{}
	metaphorTemplateTokens.ClusterName = clusterNameFlag
	metaphorTemplateTokens.CloudRegion = cloudRegionFlag
	metaphorTemplateTokens.ContainerRegistryURL = fmt.Sprintf("ghcr.io/%s/metaphor-frontend", githubOwnerFlag)
	metaphorTemplateTokens.DomainName = k3d.DomainName
	metaphorTemplateTokens.MetaphorDevelopmentIngressURL = fmt.Sprintf("metaphor-development.%s", k3d.DomainName)
	metaphorTemplateTokens.MetaphorStagingIngressURL = fmt.Sprintf("metaphor-staging.%s", k3d.DomainName)
	metaphorTemplateTokens.MetaphorProductionIngressURL = fmt.Sprintf("metaphor-production.%s", k3d.DomainName)

	//* git clone and detokenize the metaphor-frontend-template repository
	if !viper.GetBool("kubefirst-checks.metaphor-repo-pushed") {

		err := k3d.PrepareMetaphorRepository(
			config.GitProvider,
			config.DestinationMetaphorRepoGitURL,
			config.K1Dir,
			config.MetaphorDir,
			metaphorTemplateBranchFlag,
			metaphorTemplateURLFlag,
			&metaphorTemplateTokens,
		)
		if err != nil {
			return err
		}

		metaphorRepo, err := git.PlainOpen(config.MetaphorDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		err = metaphorRepo.Push(&git.PushOptions{
			RemoteName: config.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			return err
		}

		log.Info().Msgf("successfully pushed gitops to git@%s/%s/metaphor-frontend", cGitHost, cGitOwner)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("pushed detokenized metaphor-frontend repository to %s/%s", cGitHost, cGitOwner)

		viper.Set("kubefirst-checks.metaphor-repo-pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	//* create k3d resources
	if !viper.GetBool("kubefirst-checks.terraform-apply-k3d") {
		log.Info().Msg("Creating k3d cluster")

		err := k3d.ClusterCreate(clusterNameFlag, config.K1Dir, config.K3dClient, config.Kubeconfig)
		if err != nil {
			return err
		}

		log.Info().Msg("successfully created k3d cluster")
		viper.Set("kubefirst-checks.terraform-apply-k3d", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created k3d cluster resources")
	}

	clientset, err := k8s.GetClientSet(dryRunFlag, config.Kubeconfig)
	if err != nil {
		return err
	}

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := k3d.AddK3DSecrets(
			atlantisWebhookSecret,
			atlantisWebhookURL,
			viper.GetString("kbot.public-key"),
			config.DestinationGitopsRepoGitURL,
			viper.GetString("kbot.private-key"),
			false,
			config.GitProvider,
			cGitUser,
			config.Kubeconfig,
		)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already added secrets to k3d cluster")
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

	//* helm add argo repository && update
	helmRepo := helm.HelmRepo{
		RepoName:     "argo",
		RepoURL:      "https://argoproj.github.io/argo-helm",
		ChartName:    "argo-cd",
		Namespace:    "argocd",
		ChartVersion: "4.10.5",
	}

	//* helm add repo and update
	executionControl = viper.GetBool("kubefirst-checks.argocd-helm-repo-added")
	if !executionControl {
		log.Info().Msgf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL)
		helm.AddRepoAndUpdateRepo(dryRunFlag, config.HelmClient, helmRepo, config.Kubeconfig)
		log.Info().Msg("helm repo added")
		viper.Set("kubefirst-checks.argocd-helm-repo-added", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("argo helm repository already added, continuing")
	}
	//* helm install argocd
	executionControl = viper.GetBool("kubefirst-checks.argocd-helm-install")
	if !executionControl {
		log.Info().Msgf("helm install %s and wait", helmRepo.RepoName)
		// todo adopt golang helm client for helm install
		err := helm.Install(dryRunFlag, config.HelmClient, helmRepo, config.Kubeconfig)
		if err != nil {
			return err
		}
		viper.Set("kubefirst-checks.argocd-helm-install", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("argo helm already installed, continuing")
	}

	// Wait for ArgoCD StatefulSet Pods to transition to Running
	argoCDStatefulSet, err := k8s.ReturnStatefulSetObject(
		config.Kubeconfig,
		"app.kubernetes.io/part-of",
		"argocd",
		"argocd",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding ArgoCD StatefulSet: %s", err)
	}
	_, err = k8s.WaitForStatefulSetReady(config.Kubeconfig, argoCDStatefulSet, 90, false)
	if err != nil {
		log.Info().Msgf("Error waiting for ArgoCD StatefulSet ready state: %s", err)
	}

	//* ArgoCD port-forward
	argoCDStopChannel := make(chan struct{}, 1)
	defer func() {
		close(argoCDStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		config.Kubeconfig,
		"argocd-server",
		"argocd",
		8080,
		8080,
		argoCDStopChannel,
	)
	log.Info().Msgf("port-forward to argocd is available at %s", k3d.ArgocdPortForwardURL)

	var argocdPassword string
	//* argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	if !executionControl {
		log.Info().Msg("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

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
	} else {
		log.Info().Msg("argo credentials already set, continuing")
	}

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		log.Info().Msg("applying the registry application to argocd")
		registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", config.K1Dir, clusterNameFlag)
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClient, "--kubeconfig", config.Kubeconfig, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
		if err != nil {
			log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
			return err
		}
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
	}

	// Wait for Vault StatefulSet Pods to transition to Running
	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		config.Kubeconfig,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding Vault StatefulSet: %s", err)
	}
	_, err = k8s.WaitForStatefulSetReady(config.Kubeconfig, vaultStatefulSet, 60, false)
	if err != nil {
		log.Info().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
	}

	log.Info().Msg("pausing for vault to become ready...")
	time.Sleep(time.Second * 45)

	minioStopChannel := make(chan struct{}, 1)
	defer func() {
		close(minioStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		config.Kubeconfig,
		"minio",
		"minio",
		9000,
		9000,
		minioStopChannel,
	)

	//copy files to Minio
	endpoint := "localhost:9000"
	accessKeyID := "k-ray"
	secretAccessKey := "feedkraystars"

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
		Region: "us-k3d-1",
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

	log.Printf("Successfully uploaded %s to bucket %d\n", objectName, info.Bucket)

	//* vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		config.Kubeconfig,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	//* configure vault with terraform
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

		//* run vault terraform
		log.Info().Msg("configuring vault with terraform")

		tfEnvs := map[string]string{}

		tfEnvs["TF_VAR_email_address"] = "your@email.com"
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = cGitToken
		tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = "k1_local_vault_token"
		tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = "k1_local_vault_token"
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
		tfEnvs["TF_VAR_kubefirst_bot_ssh_private_key"] = viper.GetString("kbot.private-key")
		tfEnvs["TF_VAR_aws_access_key_id"] = "kray"
		tfEnvs["TF_VAR_aws_secret_access_key"] = "feedkraystars"

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
		tfEnvs[fmt.Sprintf("TF_VAR_%s_token", strings.ToUpper(config.GitProvider))] = cGitToken
		tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
		tfEnvs["TF_VAR_vault_token"] = "k1_local_vault_token"
		tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
		tfEnvs["VAULT_TOKEN"] = "k1_local_vault_token"
		tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
		tfEnvs["TF_VAR_atlantis_repo_webhook_url"] = atlantisWebhookURL
		tfEnvs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = cGitToken
		tfEnvs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = cGitOwner

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

	//PostRun string replacement
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
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: config.GitProvider,
		Auth:       publicKeys,
	})
	if err != nil {
		log.Info().Msgf("Error pushing repo: %s", err)
	}
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content post run")
	if err != nil {
		return err
	}

	// Wait for console Deployment Pods to transition to Running
	consoleDeployment, err := k8s.ReturnDeploymentObject(
		config.Kubeconfig,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding console Deployment: %s", err)
	}
	_, err = k8s.WaitForDeploymentReady(config.Kubeconfig, consoleDeployment, 120)
	if err != nil {
		log.Info().Msgf("Error waiting for console Deployment ready state: %s", err)
	}

	//* console port-forward
	consoleStopChannel := make(chan struct{}, 1)
	defer func() {
		close(consoleStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		config.Kubeconfig,
		"kubefirst-console",
		"kubefirst",
		8080,
		9094,
		consoleStopChannel,
	)

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

	reports.LocalHandoffScreenV2(argocdPassword, clusterNameFlag, githubOwnerFlag, config, dryRunFlag, false)

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(k3d.DomainName, pkg.MetricMgmtClusterInstallCompleted, k3d.CloudProvider, config.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	time.Sleep(time.Millisecond * 100) // allows progress bars to finish

	return nil
}
