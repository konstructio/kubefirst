package civo

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

// validateCivo is responsible for gathering all of the information required to execute a kubefirst civo cloud creation with github (currently)
// this function needs to provide all the generated values and provides a single space for writing and updating configuration up front.
func validateCivo(cmd *cobra.Command, args []string) error {

	//* get cli flag values for storage in `$HOME/.kubefirst`
	adminEmailFlag, err := cmd.Flags().GetString("admin-email")
	if err != nil {
		return err
	}

	domainNameFlag, err := cmd.Flags().GetString("domain-name")
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

	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
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

	silentModeFlag, err := cmd.Flags().GetBool("silent-mode")
	if err != nil {
		return err
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return err
	}

	cloudProvider := "civo"
	gitProvider := "github"

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitStarted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	//! hack
	// if err := pkg.ValidateK1Folder(config.K1FolderPath); err != nil {
	// 	return err
	// }

	// this branch flag value is overridden with a tag when running from a
	// kubefirst binary for version compatibility
	if gitopsTemplateBranchFlag == "main" && configs.K1Version != "development" {
		gitopsTemplateBranchFlag = configs.K1Version
	}
	log.Info().Msg(fmt.Sprintf("kubefirst version configs.K1Version: %s ", configs.K1Version))
	log.Info().Msg(fmt.Sprintf("cloning gitops-template repo url: %s ", gitopsTemplateURLFlag))
	log.Info().Msg(fmt.Sprintf("cloning gitops-template repo branch: %s ", gitopsTemplateBranchFlag))
	// this branch flag value is overridden with a tag when running from a
	// kubefirst binary for version compatibility
	if metaphorTemplateBranchFlag == "main" && configs.K1Version != "development" {
		metaphorTemplateBranchFlag = configs.K1Version
	}

	log.Info().Msg(fmt.Sprintf("cloning metaphor template url: %s ", metaphorTemplateURLFlag))
	log.Info().Msg(fmt.Sprintf("cloning metaphor template branch: %s ", metaphorTemplateBranchFlag))

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
		return err
	}

	// todo waiting for johns response
	// todo this clusterId need to go to state store,
	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}

	k1DirPath := fmt.Sprintf("%s/.k1", homePath)

	// todo validate flags
	viper.Set("admin-email", adminEmailFlag)
	viper.Set("argocd.helm.chart-version", "4.10.5")
	viper.Set("argocd.local.service", "http://localhost:8080")
	viper.Set("vault.local.service", "http://localhost:8200")
	viper.Set("cloud-provider", cloudProvider)
	viper.Set("git-provider", gitProvider)
	viper.Set("kubefirst.k1-dir", k1DirPath)
	viper.Set("kubefirst.k1-tools-dir", fmt.Sprintf("%s/tools", k1DirPath))
	viper.Set("kubefirst.k1-gitops-dir", fmt.Sprintf("%s/gitops", k1DirPath))
	viper.Set("kubefirst.k1-metaphor-dir", fmt.Sprintf("%s/metaphor-frontend", k1DirPath))
	viper.Set("kubefirst.helm-client-path", fmt.Sprintf("%s/tools/helm", k1DirPath))
	viper.Set("kubefirst.helm-client-version", "v3.6.1")
	viper.Set("kubefirst.kubeconfig-path", fmt.Sprintf("%s/kubeconfig", k1DirPath))
	viper.Set("kubefirst.kubectl-client-path", fmt.Sprintf("%s/tools/kubectl", k1DirPath))
	viper.Set("kubefirst.kubectl-client-version", "v1.23.15") // todo make configs like this more discoverable in struct?
	viper.Set("kubefirst.kubefirst-config-path", fmt.Sprintf("%s/%s", homePath, ".kubefirst"))
	viper.Set("kubefirst.terraform-client-path", fmt.Sprintf("%s/tools/terraform", k1DirPath))
	viper.Set("kubefirst.terraform-client-version", "1.0.11")
	viper.Set("localhost.os", runtime.GOOS)
	viper.Set("localhost.architecture", runtime.GOARCH)
	viper.Set("github.atlantis.webhook.secret", pkg.Random(20))
	viper.Set("github.atlantis.webhook.url", fmt.Sprintf("https://atlantis.%s/events", domainNameFlag))
	viper.Set("github.repo.gitops.url", fmt.Sprintf("https://github.com/%s/gitops.git", githubOwnerFlag))
	viper.Set("github.repo.metaphor-frontend.url", fmt.Sprintf("https://github.com/%s/metaphor-frontend.git", githubOwnerFlag))
	githubOwnerRootGitURL := fmt.Sprintf("git@github.com:%s", githubOwnerFlag)
	viper.Set("github.repo.gitops.giturl", fmt.Sprintf("%s/gitops.git", githubOwnerRootGitURL))
	viper.Set("github.repo.metaphor-frontend.giturl", fmt.Sprintf("%s/metaphor-frontend.git", githubOwnerRootGitURL))
	viper.Set("template-repo.gitops.branch", gitopsTemplateBranchFlag)
	viper.Set("template-repo.gitops.url", gitopsTemplateURLFlag)
	viper.Set("template-repo.metaphor-frontend.url", metaphorTemplateURLFlag)
	viper.Set("template-repo.metaphor-frontend.branch", metaphorTemplateBranchFlag)
	viper.Set("vault.token", "k1_local_vault_token")

	kubefirstStateStoreBucketName := fmt.Sprintf("k1-state-store-%s-%s", clusterNameFlag, clusterId)

	viper.WriteConfig()

	pkg.InformUser("checking authentication to required providers", silentModeFlag)

	//* CIVO START
	executionControl := viper.GetBool("kubefirst.checks.civo.complete")
	if !executionControl {
		civoToken := viper.GetString("civo.token")
		if os.Getenv("CIVO_TOKEN") != "" {
			civoToken = os.Getenv("CIVO_TOKEN")
		}

		if civoToken == "" {
			fmt.Println("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security to retrieve your token\nand enter it here, then press Enter:")
			civoToken, err := terminal.ReadPassword(0)
			if err != nil {
				return errors.New("error reading password input from user")
			}

			os.Setenv("CIVO_TOKEN", string(civoToken))
			log.Info().Msg("CIVO_TOKEN set - continuing")
		}
		viper.Set("kubefirst.checks.civo.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed civo token check - continuing")
	}

	executionControl = viper.GetBool("civo.object-storage-creds.complete")
	if !executionControl {
		creds, err := civo.GetAccessCredentials(kubefirstStateStoreBucketName, cloudRegionFlag)
		if err != nil {
			log.Info().Msg(err.Error())
		}
		viper.Set("civo.object-storage-creds.access-key-id", creds.AccessKeyID)
		viper.Set("civo.object-storage-creds.secret-access-key-id", creds.SecretAccessKeyID)
		viper.Set("civo.object-storage-creds.name", creds.Name)
		viper.Set("civo.object-storage-creds.id", creds.ID)
		viper.Set("civo.object-storage-creds.complete", true)
		viper.WriteConfig()
		log.Info().Msg("civo object storage credentials created and set")
	} else {
		log.Info().Msg("already created civo object storage credentials - continuing")
	}

	executionControl = viper.GetBool("kubefirst.state-store-bucket.complete")
	if !executionControl {
		accessKeyId := viper.GetString("civo.object-storage-creds.access-key-id")
		log.Info().Msgf("access key id %s", accessKeyId)

		bucket, err := civo.CreateStorageBucket(accessKeyId, kubefirstStateStoreBucketName, cloudRegionFlag)
		if err != nil {
			log.Info().Msg(err.Error())
			return err
		}

		viper.Set("civo.object-storage-bucket.id", bucket.ID)
		viper.Set("civo.object-storage-bucket.name", bucket.Name)
		viper.Set("kubefirst.state-store-bucket.complete", true)
		viper.WriteConfig()
		log.Info().Msg("civo state store bucket created")
	} else {
		log.Info().Msg("already created civo state store bucket - continuing")
	}
	//* CIVO END

	executionControl = viper.GetBool("kubefirst.checks.github.complete")
	if !executionControl {

		httpClient := http.DefaultClient
		githubToken := os.Getenv("GITHUB_TOKEN")
		if len(githubToken) == 0 {
			return errors.New("ephemeral tokens not supported for cloud installations, please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
		}
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		// get github data to set user based on the provided token
		log.Info().Msg("verifying github authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(githubToken)
		if err != nil {
			return err
		}

		err = gitHubHandler.CheckGithubOrganizationPermissions(githubToken, githubOwnerFlag, githubUser)
		if err != nil {
			return err
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
		viper.Set("github.owner", githubOwnerFlag)
		viper.Set("github.user", githubUser)
		viper.Set("kubefirst.checks.github.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed github checks - continuing")
	}

	executionControl = viper.GetBool("kubefirst.checks.bot-setup.complete")
	if !executionControl {

		log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
		sshPrivateKey, sshPublicKey, err := ssh.CreateSshKeyPair()
		if err != nil {
			return err
		}
		if len(kbotPasswordFlag) == 0 {
			kbotPasswordFlag = pkg.Random(20)
		}
		log.Info().Msg("ssh key pair creation complete")

		viper.Set("kubefirst.telemetry", useTelemetryFlag)
		viper.Set("kubefirst.cluster-name", clusterNameFlag)
		viper.Set("kubefirst.cluster-type", clusterTypeFlag)
		viper.Set("domain-name", domainNameFlag)
		viper.Set("cloud-region", cloudRegionFlag)

		viper.Set("kubefirst.bot.password", kbotPasswordFlag)
		viper.Set("kubefirst.bot.private-key", sshPrivateKey)
		viper.Set("kubefirst.bot.public-key", sshPublicKey)
		viper.Set("kubefirst.bot.user", "kbot")
		viper.Set("kubefirst.checks.bot-setup.complete", true)
		viper.WriteConfig()
		log.Info().Msg("kubefirst values and bot-setup complete")
		// todo, is this a hangover from initial gitlab? do we need this?
		log.Info().Msg("creating argocd-init-values.yaml for initial install")
		//* ex: `git@github.com:kubefirst` this is allows argocd access to the github organization repositories
		err = ssh.WriteGithubArgoCdInitValuesFile(githubOwnerRootGitURL, sshPrivateKey)
		if err != nil {
			return err
		}
		log.Info().Msg("argocd-init-values.yaml creation complete")
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitCompleted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	// todo progress bars
	// time.Sleep(time.Millisecond * 100) // to allow progress bars to finish

	return nil
}
