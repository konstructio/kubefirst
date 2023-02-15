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
	"golang.org/x/term"
)

// validateCivo is responsible for gathering all of the information required to execute a kubefirst civo cloud creation with github (currently)
// this function needs to provide all the generated values and provides a single space for writing and updating configuration up front.
func validateCivo(cmd *cobra.Command, args []string) error {

	adminEmailFlag, err := cmd.Flags().GetString("admin-email")
	if err != nil {
		return err
	} else if adminEmailFlag == "" {
		return errors.New("admin-email flag cannot be empty")
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
	} else if domainNameFlag == "" {
		return errors.New("domain-name flag cannot be empty")
	}

	githubOwnerFlag, err := cmd.Flags().GetString("github-owner")
	if err != nil {
		return err
	} else if githubOwnerFlag == "" {
		return errors.New("github-owner flag cannot be empty")
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

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}

	cloudProvider := "civo"
	gitProvider := "github"

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitStarted, cloudProvider, gitProvider); err != nil {
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

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
		return err
	}

	k1Dir := fmt.Sprintf("%s/.k1", homePath)

	viper.Set("flags.admin-email", adminEmailFlag)
	viper.Set("flags.cloud-provider", cloudProvider)
	viper.Set("flags.cloud-region", cloudRegionFlag)
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.cluster-type", clusterTypeFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.git-provider", gitProvider)
	viper.Set("flags.gitops-template-branch", gitopsTemplateBranchFlag)
	viper.Set("flags.gitops-template-url", gitopsTemplateURLFlag)
	viper.Set("flags.metaphor-template-branch", metaphorTemplateBranchFlag)
	viper.Set("flags.metaphor-template-url", metaphorTemplateURLFlag)
	viper.Set("flags.use-telemetry", useTelemetryFlag)

	viper.Set("components.argocd.helm-chart-version", "4.10.5")
	viper.Set("components.argocd.port-forward-url", "http://localhost:8080")
	viper.Set("components.atlantis.webhook.secret", pkg.Random(20))
	viper.Set("components.atlantis.webhook.url", fmt.Sprintf("https://atlantis.%s/events", domainNameFlag))
	viper.Set("components.vault.port-forward-url", "http://localhost:8200")
	viper.Set("components.vault.token", "k1_local_vault_token")

	viper.Set("github.host", "github.com")
	viper.Set("github.owner", githubOwnerFlag)
	viper.Set("github.repos.gitops.url", fmt.Sprintf("https://github.com/%s/gitops.git", githubOwnerFlag))
	viper.Set("github.repos.metaphor.url", fmt.Sprintf("https://github.com/%s/metaphor-frontend.git", githubOwnerFlag))
	githubOwnerRootGitURL := fmt.Sprintf("git@github.com:%s", githubOwnerFlag)
	viper.Set("github.repos.gitops.git-url", fmt.Sprintf("%s/gitops.git", githubOwnerRootGitURL))
	viper.Set("github.repos.metaphor.git-url", fmt.Sprintf("%s/metaphor-frontend.git", githubOwnerRootGitURL))

	viper.Set("k1-paths.gitops-dir", fmt.Sprintf("%s/gitops", k1Dir))
	viper.Set("k1-paths.helm-client", fmt.Sprintf("%s/tools/helm", k1Dir))
	viper.Set("k1-paths.k1-dir", k1Dir)
	viper.Set("k1-paths.kubeconfig", fmt.Sprintf("%s/kubeconfig", k1Dir))
	viper.Set("k1-paths.kubectl-client", fmt.Sprintf("%s/tools/kubectl", k1Dir))
	viper.Set("k1-paths.kubefirst-config", fmt.Sprintf("%s/%s", homePath, ".kubefirst"))
	viper.Set("k1-paths.logs-dir", fmt.Sprintf("%s/logs", k1Dir))
	viper.Set("k1-paths.metaphor-dir", fmt.Sprintf("%s/metaphor-frontend", k1Dir))
	viper.Set("k1-paths.terraform-client", fmt.Sprintf("%s/tools/terraform", k1Dir))
	viper.Set("k1-paths.tools-dir", fmt.Sprintf("%s/tools", k1Dir))

	viper.Set("kubefirst.cluster-id", clusterId)

	viper.Set("tools.helm.client-version", "v3.6.1")
	viper.Set("tools.kubectl.client-version", "v1.23.15")
	viper.Set("tools.terraform.client-version", "1.3.8")
	viper.Set("tools.localhost.os", runtime.GOOS)
	viper.Set("tools.localhost.architecture", runtime.GOARCH)

	kubefirstStateStoreBucketName := fmt.Sprintf("k1-state-store-%s-%s", clusterNameFlag, clusterId)

	viper.WriteConfig()

	log.Info().Msg("checking authentication to required providers")

	//* CIVO START
	executionControl := viper.GetBool("kubefirst-checks.cloud-credentials")
	if !executionControl {
		civoToken := viper.GetString("civo.token")
		if os.Getenv("CIVO_TOKEN") != "" {
			civoToken = os.Getenv("CIVO_TOKEN")
		}

		if civoToken == "" {
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
	} else {
		log.Info().Msg("already completed cloud credentials check - continuing")
	}

	executionControl = viper.GetBool("kubefirst-checks.state-store-creds")
	if !executionControl {
		creds, err := civo.GetAccessCredentials(kubefirstStateStoreBucketName, cloudRegionFlag)
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
	} else {
		log.Info().Msg("already created civo object storage credentials - continuing")
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
	} else {
		log.Info().Msg("skipping domain check")
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
	} else {
		log.Info().Msg("already created civo state store bucket - continuing")
	}
	//* CIVO END

	executionControl = viper.GetBool("kubefirst-checks.github-credentials")
	if !executionControl {

		httpClient := http.DefaultClient
		githubToken := os.Getenv("GITHUB_TOKEN")
		if len(githubToken) == 0 {
			return errors.New("please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
		}
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		// get github data to set user based on the provided token
		log.Info().Msg("verifying github authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(githubToken)
		if err != nil {
			return err
		}
		viper.Set("github.user", githubUser)
		viper.WriteConfig()

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

		viper.Set("kubefirst-checks.github-credentials", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed github checks - continuing")
	}

	executionControl = viper.GetBool("kubefirst-checks.kbot-setup")
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

		viper.Set("kbot.password", kbotPasswordFlag)
		viper.Set("kbot.private-key", sshPrivateKey)
		viper.Set("kbot.public-key", sshPublicKey)
		viper.Set("kbot.username", "kbot")
		viper.Set("kubefirst-checks.kbot-setup", true)
		viper.WriteConfig()
		log.Info().Msg("kbot-setup complete")
	}

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitCompleted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}
	os.Exit(1)

	// todo progress bars
	// time.Sleep(time.Millisecond * 100) // to allow progress bars to finish

	return nil
}
