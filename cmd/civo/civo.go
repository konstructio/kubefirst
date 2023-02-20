package civo

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/term"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/services"
	internalssh "github.com/kubefirst/kubefirst/internal/ssh"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runCivo(cmd *cobra.Command, args []string) error {

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 6)
	progressPrinter.AddTracker("platform-create", "Creating your kubefirst platform", 13)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// alertsEmailFlag, err := cmd.Flags().GetString("alerts-email")
	// if err != nil {
	// 	return err
	// }

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

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		return err
	}

	// required for destroy command
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.dry-run", dryRunFlag)
	viper.Set("flags.github-owner", githubOwnerFlag)
	viper.WriteConfig()

	config := civo.GetConfig(clusterNameFlag, domainNameFlag, githubOwnerFlag)

	var sshPrivateKey, sshPublicKey string

	// todo placed in configmap in kubefirst namespace, included in telemetry
	clusterId := viper.GetString("kubefirst.cluster-id")
	if clusterId == "" {
		clusterId = pkg.GenerateClusterID()
		viper.Set("kubefirst.cluster-id", clusterId)
		viper.WriteConfig()
	}

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitStarted, civo.CloudProvider, civo.GitProvider); err != nil {
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

	kubefirstStateStoreBucketName := fmt.Sprintf("k1-state-store-%s-%s", clusterNameFlag, clusterId)

	log.Info().Msg("checking authentication to required providers")

	//* CIVO START
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
	quotaMessage, quotaFailures, quotaWarnings, err := returnCivoQuotaEvaluation(cloudRegionFlag, false)
	if err != nil {
		return err
	}
	switch {
	case quotaFailures > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
		return errors.New("At least one of your Civo quotas is close to its limit. Please check the error message above for additional details.")
	case quotaWarnings > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
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
		progressPrinter.IncrementTracker("preflight-checks", 1)
	} else {
		log.Info().Msg("already completed github checks - continuing")
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
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricInitCompleted, civo.CloudProvider, civo.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	//* generate public keys for ssh
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallStarted, civo.CloudProvider, civo.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	publicKeys, err := ssh.NewPublicKeys("git", []byte(sshPrivateKey), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallStarted, civo.CloudProvider, civo.GitProvider); err != nil {
			log.Info().Msg(err.Error())
		}
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		log.Info().Msg("installing kubefirst dependencies")

		err := downloadManager.CivoDownloadTools(
			config.HelmClient,
			civo.HelmClientVersion,
			config.KubectlClient,
			civo.KubectlClientVersion,
			civo.LocalhostOS,
			civo.LocalhostArch,
			config.TerraformClient,
			config.ToolsDir,
		)
		if err != nil {
			return err
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
	}

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		log.Info().Msg("generating your new gitops repository")
		gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranchFlag, config.GitopsDir, gitopsTemplateURLFlag)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}
		log.Info().Msg("gitops repository clone complete")

		err = civo.CivoGithubAdjustGitopsTemplateContent(civo.CloudProvider, clusterNameFlag, clusterTypeFlag, civo.GitProvider, config.K1Dir, config.GitopsDir)
		if err != nil {
			return err
		}

		civo.DetokenizeCivoGithubGitops(config.GitopsDir)
		if err != nil {
			return err
		}
		err = gitClient.AddRemote(config.DestinationGitopsRepoGitURL, civo.GitProvider, gitopsRepo)
		if err != nil {
			return err
		}

		err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
		if err != nil {
			return err
		}

		// todo emit init telemetry end
		viper.Set("kubefirst-checks.gitops-ready-to-push", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
	}

	//* create teams and repositories in github
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-github")
	if !executionControl {
		log.Info().Msg("Creating github resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/github"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating github resources with terraform %s : %s", tfEntrypoint, err))
		}

		log.Info().Msgf("Created git repositories and teams in github.com/%s", githubOwnerFlag)
		viper.Set("kubefirst-checks.terraform-apply-github", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already created github terraform resources")
		progressPrinter.IncrementTracker("platform-create", 1)
	}

	//* push detokenized gitops-template repository content to new remote
	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		gitopsRepo, err := git.PlainOpen(config.GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: civo.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", config.DestinationGitopsRepoGitURL)
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/gitops", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("Created git repositories and teams in github.com/%s", githubOwnerFlag)
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
		progressPrinter.IncrementTracker("platform-create", 1)
	}

	//* git clone and detokenize the metaphor-frontend-template repository
	if !viper.GetBool("kubefirst-checks.metaphor-repo-pushed") {

		if configs.K1Version != "" {
			gitopsTemplateBranchFlag = configs.K1Version
		}

		log.Info().Msg("generating your new metaphor-frontend repository")
		metaphorRepo, err := gitClient.CloneRefSetMain(metaphorTemplateBranchFlag, config.MetaphorDir, metaphorTemplateURLFlag)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", config.MetaphorDir)
		}

		log.Info().Msg("metaphor repository clone complete")

		err = civo.CivoGithubAdjustMetaphorTemplateContent(civo.GitProvider, config.K1Dir, config.MetaphorDir)
		if err != nil {
			return err
		}

		err = civo.DetokenizeCivoGithubMetaphor(config.MetaphorDir)
		if err != nil {
			return err
		}
		err = gitClient.AddRemote(config.DestinationMetaphorRepoGitURL, civo.GitProvider, metaphorRepo)
		if err != nil {
			return err
		}

		err = gitClient.Commit(metaphorRepo, "committing detokenized metaphor-frontend-template repo content")
		if err != nil {
			return err
		}

		err = metaphorRepo.Push(&git.PushOptions{
			RemoteName: civo.GitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", config.DestinationMetaphorRepoGitURL)
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/metaphor-frontend", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		log.Info().Msgf("pushed detokenized metaphor-frontend repository to github.com/%s", githubOwnerFlag)

		viper.Set("kubefirst-checks.metaphor-repo-pushed", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
	}

	//* create civo cloud resources
	if !viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		log.Info().Msg("Creating civo cloud resources with terraform")

		tfEntrypoint := config.GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating civo resources with terraform %s : %s", tfEntrypoint, err))
		}

		log.Info().Msg("Created civo cloud resources")
		viper.Set("kubefirst-checks.terraform-apply-civo", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already created github terraform resources")
		progressPrinter.IncrementTracker("platform-create", 1)
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
		err := civo.BootstrapCivoMgmtCluster(dryRunFlag, config.Kubeconfig)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already added secrets to civo cluster")
		progressPrinter.IncrementTracker("platform-create", 1)
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
	} else {
		log.Info().Msg("no files found in secrets directory, continuing")
	}

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
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("argo helm repository already added, continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
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
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("argo helm already installed, continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
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
	_, err = k8s.WaitForStatefulSetReady(config.Kubeconfig, argoCDStatefulSet, 90)
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
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("argo credentials already set, continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
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
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("argocd registry create already done, continuing")
		progressPrinter.IncrementTracker("platform-create", 1)
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
	_, err = k8s.WaitForStatefulSetReady(config.Kubeconfig, vaultStatefulSet, 60)
	if err != nil {
		log.Info().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
	}

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

		tfEnvs = civo.GetVaultTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEntrypoint := config.GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}

		log.Info().Msg("vault terraform executed successfully")
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already executed vault terraform")
		progressPrinter.IncrementTracker("platform-create", 1)
	}

	//* create users
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-users")
	if !executionControl {
		log.Info().Msg("applying users terraform")

		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetUsersTerraformEnvs(tfEnvs)
		tfEntrypoint := config.GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(dryRunFlag, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}
		log.Info().Msg("executed users terraform successfully")
		// progressPrinter.IncrementTracker("step-users", 1)
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
		progressPrinter.IncrementTracker("platform-create", 1)
	} else {
		log.Info().Msg("already created users with terraform")
		progressPrinter.IncrementTracker("platform-create", 1)
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
	log.Info().Msg("welcome to your new kubefirst platform powered by Civo cloud")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	reports.LocalHandoffScreen(dryRunFlag, false)

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallCompleted, civo.CloudProvider, civo.GitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	time.Sleep(time.Millisecond * 100) // allows progress bars to finish

	return nil
}
