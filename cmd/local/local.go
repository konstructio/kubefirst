package local

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/wrappers"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/metaphor"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	useTelemetry   bool
	dryRun         bool
	silentMode     bool
	enableConsole  bool
	skipMetaphor   bool
	gitOpsBranch   string
	gitOpsRepo     string
	gitOpsOrg      string
	metaphorBranch string
	adminEmail     string
	templateTag    string
	logLevel       string

	// ngrok context that is used to control ngrok context cancellation, and is called at the end of the installation,
	// after the user closes Kubefirst installer.
	cancelContext context.CancelFunc
)

func NewCommand() *cobra.Command {

	localCmd := &cobra.Command{
		Use:      "local",
		Short:    "Kubefirst localhost installation",
		Long:     "Kubefirst localhost enable a localhost installation without the requirement of a cloud provider.",
		PreRunE:  validateLocal,
		RunE:     runLocal,
		PostRunE: runPostLocal,
	}

	localCmd.Flags().BoolVar(&useTelemetry, "use-telemetry", true, "installer won't send telemetry data if --use-telemetry=false is set")
	localCmd.Flags().BoolVar(&dryRun, "dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	localCmd.Flags().BoolVar(&silentMode, "silent", false, "enable silentMode mode will make the UI return less content to the screen")
	localCmd.Flags().BoolVar(&enableConsole, "enable-console", true, "If hand-off screen will be presented on a browser UI")
	localCmd.Flags().BoolVar(&skipMetaphor, "skip-metaphor", false, "If metaphor application suite must be skiped to deploy")
	// todo: get it from GH token , use it for console
	localCmd.Flags().StringVar(&adminEmail, "admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	localCmd.Flags().StringVar(&metaphorBranch, "metaphor-branch", "", "metaphor application branch")
	localCmd.Flags().StringVar(&gitOpsBranch, "gitops-branch", "", "version/branch used on git clone")
	localCmd.Flags().StringVar(&gitOpsRepo, "gitops-repo", "gitops", "Prefix of the repo for gitops template, repo name has -template")
	localCmd.Flags().StringVar(&gitOpsOrg, "gitops-org", "kubefirst", "Helpful when using forks of gitops for testing")
	localCmd.Flags().StringVar(&templateTag, "template-tag", "",
		"when running a built version, and ldflag is set for the Kubefirst version, it will use this tag value to clone the templates (gitops and metaphor's)",
	)
	localCmd.Flags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"available log levels are: trace, debug, info, warning, error, fatal, panic",
	)

	// on error, doesnt show helper/usage
	localCmd.SilenceUsage = true

	// wire up new commands
	localCmd.AddCommand(NewDestroyCommand())

	return localCmd
}

func runLocal(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()

	gitProvider := "github"
	cloud := "k3d"

	progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
	progressPrinter.AddTracker("step-base", "Setup base cluster", 2)
	progressPrinter.AddTracker("step-apps", "Install apps to cluster", 4)
	if useTelemetry {
		if err := wrappers.SendSegmentIoTelemetry("", pkg.MetricMgmtClusterInstallStarted, cloud, gitProvider); err != nil {
			log.Error().Err(err).Msg("")
		}
		log.Info().Msg("Telemetry info sent")
	} else {
		pkg.InformUser("Telemetry skipped by user request", silentMode)
	}

	// todo need to add go channel to control when ngrok should close
	// and use context to handle closing the open goroutine/connection
	//go pkg.RunNgrok(context.TODO(), pkg.LocalAtlantisURL)

	if !viper.GetBool("kubefirst.done") {
		if gitProvider == "github" {
			log.Info().Msg("Installing Github version of Kubefirst")
			viper.Set("git.mode", "github")
			err := k3d.CreateK3dCluster()
			if err != nil {
				return err
			}
		}
		viper.Set("kubefirst.done", true)
		viper.WriteConfig()
	}

	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), silentMode)

	executionControl := viper.GetBool("terraform.github.apply.complete")
	// create github teams in the org and gitops repo
	if !executionControl {
		pkg.InformUser("Creating github resources with terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, map[string]string{}) // todo need to get envs

		pkg.InformUser(fmt.Sprintf("Created gitops Repo in github.com/%s", viper.GetString("github.owner")), silentMode)
		progressPrinter.IncrementTracker("step-github", 1)
		viper.Set("terraform.github.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created github terraform resources")
	}

	// push our locally detokenized gitops repo to remote github
	githubHost := viper.GetString("github.host")
	githubOwner := viper.GetString("github.owner")
	localRepo := "gitops"
	remoteName := "github"
	executionControl = viper.GetBool("github.gitops.hydrated") // todo fix this executionControl value `github.detokenized-gitops.pushed`?
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("pushing local detokenized gitops content to new remote github.com/%s", viper.GetString("github.owner")), silentMode)
		gitClient.PushLocalRepoToEmptyRemote(githubHost, githubOwner, localRepo, remoteName)
	} else {
		log.Info().Msg("already hydrated the github gitops repository")
	}
	progressPrinter.IncrementTracker("step-github", 1)

	// create kubernetes cluster
	executionControl = viper.GetBool("k3d.created")
	if !executionControl {
		pkg.InformUser("Creating K8S Cluster", silentMode)
		err := k3d.CreateK3dCluster()
		if err != nil {
			log.Error().Err(err).Msg("Error installing k3d cluster")
			return err
		}
	} else {
		log.Info().Msg("already created k3d cluster")
	}
	progressPrinter.IncrementTracker("step-base", 1)
	progressPrinter.IncrementTracker("step-github", 1)

	//
	// create local certs using MkCert tool
	//

	//TODO: Verify Approach on PR
	// We will not install CAROOT from mkcert into users machine as it requires sudo access.
	// ssl.InstallCALocal(config)
	log.Info().Msg("we will use mkcert for creating certificates with a common root cert")
	log.Info().Msg("the certificates are by default at:  $HOME/.local/share/mkcert/")
	log.Info().Msgf("if you have sudo access you can run: %s -install", config.MkCertPath)
	log.Info().Msg("that will update your trust store with mkcert rootCA, allowing your browser to trust this installation certs")
	log.Info().Msg("learn more at: https://github.com/FiloSottile/mkcert#changing-the-location-of-the-ca-files")
	log.Info().Msg("creating local certificates")
	if err := ssl.CreateCertificatesForLocalWrapper(config); err != nil {
		log.Error().Err(err).Msg("")
	}
	log.Info().Msg("creating local certificates done")

	// add secrets to cluster
	// todo there is a secret condition in AddK3DSecrets to this not checked
	executionControl = viper.GetBool("kubernetes.vault.secret.created")
	if !executionControl {
		err := k3d.AddK3DSecrets(dryRun, config.KubeConfigPath)
		if err != nil {
			log.Error().Err(err).Msg("Error AddK3DSecrets")
			return err
		}
	} else {
		log.Info().Msg("already added secrets to k3d cluster")
	}

	log.Info().Msg("storing certificates into application secrets namespace")
	if err := k8s.CreateSecretsFromCertificatesForLocalWrapper(config); err != nil {
		log.Error().Err(err).Msg("")
	}
	log.Info().Msg("storing certificates into application secrets namespace done")

	// create argocd initial repository config
	executionControl = viper.GetBool("argocd.initial-repository.created")
	if !executionControl {
		pkg.InformUser("create initial argocd repository", silentMode)
		// Enterprise users need to be able to set the hostname for git.
		gitOpsRepo := fmt.Sprintf("git@%s:%s/gitops.git", viper.GetString("github.host"), viper.GetString("github.owner"))

		argoCDConfig := argocd.GetArgoCDInitialLocalConfig(
			gitOpsRepo,
			viper.GetString("botprivatekey"),
		)

		err := argocd.CreateInitialArgoCDRepository(argoCDConfig, config.KubeConfigPath)
		if err != nil {
			log.Error().Err(err).Msg("Error CreateInitialArgoCDRepository")
			return err
		}
	} else {
		log.Info().Msg("already created initial argocd repository")
	}

	// helm add argo repository && update
	helmRepo := helm.HelmRepo{
		RepoName:     pkg.HelmRepoName,
		RepoURL:      pkg.HelmRepoURL,
		ChartName:    pkg.HelmRepoChartName,
		Namespace:    pkg.HelmRepoNamespace,
		ChartVersion: pkg.HelmRepoChartVersion,
	}

	executionControl = viper.GetBool("argocd.helm.repo.updated")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL), silentMode)
		helm.AddRepoAndUpdateRepo(dryRun, config.HelmClientPath, helmRepo, config.KubeConfigPath)
	}

	// helm install argocd
	// todo undo this is from vault-spike
	executionControl = viper.GetBool("argocd.helm.install.complete")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), silentMode)
		helm.Install(dryRun, config.HelmClientPath, helmRepo, config.KubeConfigPath)
	}
	progressPrinter.IncrementTracker("step-apps", 1)

	// argocd pods are running
	executionControl = viper.GetBool("argocd.ready")
	if !executionControl {
		argocd.WaitArgoCDToBeReady(dryRun, config.KubeConfigPath, config.KubectlClientPath)
		pkg.InformUser("ArgoCD is running, continuing", silentMode)
	} else {
		log.Info().Msg("already waited for argocd to be ready")
	}

	// argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("argocd.credentials.set")
	if !executionControl {
		pkg.InformUser("Setting argocd username and password credentials", silentMode)
		k8s.SetArgocdCreds(dryRun, config.KubeConfigPath)
		pkg.InformUser("argocd username and password credentials set successfully", silentMode)

		pkg.InformUser("Getting an argocd auth token", silentMode)
		_ = argocd.GetArgocdAuthToken(dryRun)
		pkg.InformUser("argocd admin auth token set", silentMode)

		viper.Set("argocd.credentials.set", true)
		viper.WriteConfig()
	}

	// argocd sync registry and start sync waves
	executionControl = viper.GetBool("argocd.registry.applied")
	if !executionControl {
		pkg.InformUser("applying the registry application to argocd", silentMode)
		registryYamlPath := fmt.Sprintf("%s/gitops/registry.yaml", config.K1FolderPath)
		err := argocd.KubectlCreateApplication(config.KubeConfigPath, config.KubectlClientPath, registryYamlPath)
		if err != nil {
			log.Error().Err(err).Msg("Error applying registry application to argocd")
			return err
		}
	}

	progressPrinter.IncrementTracker("step-apps", 1)

	// vault in running state
	executionControl = viper.GetBool("vault.status.running")
	if !executionControl {
		pkg.InformUser("waiting for Vault to be ready...", silentMode)
		vault.WaitVaultToBeRunning(dryRun, config.KubeConfigPath, config.KubectlClientPath)
	}

	k8s.LoopUntilPodIsReady(dryRun, config.KubeConfigPath, config.KubectlClientPath)

	// configure vault with terraform
	executionControl = viper.GetBool("terraform.vault.apply.complete")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)
		//* set known vault token
		viper.Set("vault.token", "k1_local_vault_token")
		viper.WriteConfig()

		//* run vault terraform
		pkg.InformUser("configuring vault with terraform", silentMode)
		tfEntrypoint := config.GitOpsRepoPath + "/terraform/vault"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, map[string]string{}) // todo need to get envs

		pkg.InformUser("vault terraform executed successfully", silentMode)

		//* create vault configurerd secret
		// todo remove this code
		log.Info().Msg("creating vault configured secret")
		k8s.CreateVaultConfiguredSecret(dryRun, config.KubeConfigPath, config.KubectlClientPath)
		pkg.InformUser("Vault secret created", silentMode)
	} else {
		log.Info().Msg("already executed vault terraform")
	}

	// create users
	executionControl = viper.GetBool("terraform.users.apply.complete")
	if !executionControl {
		pkg.InformUser("applying users terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/users"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, map[string]string{}) // todo need to get envs

		pkg.InformUser("executed users terraform successfully", silentMode)
		// progressPrinter.IncrementTracker("step-users", 1)
	} else {
		log.Info().Msg("already created users with terraform")
	}

	// TODO: K3D =>  NEED TO REMOVE local-backend.tf and rename remote-backend.md

	pkg.InformUser("To use your cluster port-forward - argocd", silentMode)
	pkg.InformUser("If not automatically injected, your kubeconfig is at:", silentMode)
	pkg.InformUser("k3d kubeconfig get "+viper.GetString("cluster-name"), silentMode)
	pkg.InformUser("Expose Argo-CD", silentMode)
	pkg.InformUser("kubectl -n argocd port-forward svc/argocd-server 8080:80", silentMode)
	pkg.InformUser("Argo User: "+viper.GetString("argocd.admin.username"), silentMode)
	pkg.InformUser("Argo Password: "+viper.GetString("argocd.admin.password"), silentMode)

	progressPrinter.IncrementTracker("step-apps", 1)
	progressPrinter.IncrementTracker("step-base", 1)
	progressPrinter.IncrementTracker("step-apps", 1)

	if !viper.GetBool("chartmuseum.host.resolved") {

		pkg.AwaitHostNTimes(config.ChartmuseumLocalURL+"/health", 5, 5)
		viper.Set("chartmuseum.host.resolved", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already resolved host for chartmuseum, continuing")
	}

	pkg.InformUser("Deploying metaphor applications", silentMode)
	err := metaphor.DeployMetaphorGithubLocal(dryRun, skipMetaphor, githubOwner, metaphorBranch, configs.K1Version)
	if err != nil {
		pkg.InformUser("Error deploy metaphor applications", silentMode)
		log.Error().Err(err).Msg("Error running deployMetaphorCmd")
	}

	// update terraform s3 backend to internal k8s dns (s3/minio bucket)
	err = pkg.UpdateTerraformS3BackendForK8sAddress(config.K1FolderPath)
	if err != nil {
		return err
	}

	// create a new branch and push changes
	branchName := "update-s3-backend"
	branchNameRef := plumbing.ReferenceName("refs/heads/" + branchName)

	// force update cloned gitops-template terraform files to use Minio backend
	err = gitClient.UpdateLocalTerraformFilesAndPush(
		githubHost,
		githubOwner,
		config.K1FolderPath,
		localRepo,
		remoteName,
		branchNameRef,
	)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	log.Info().Msg("sleeping after git commit with Minio backend update for Terraform")
	time.Sleep(3 * time.Second)

	// create a PR, atlantis will identify it's a Terraform change/file update and trigger atlantis plan
	// it's a goroutine since it can run in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		pkg.InformUser(`waiting "atlantis plan" finish to proceed...`, silentMode)
		gitHubClient := githubWrapper.New()

		base := "main"
		title := "update S3 backend to minio / internal k8s dns"
		body := "use internal Kubernetes dns"
		gitHubUser := viper.GetString("github.user")
		pullRequest, err := gitHubClient.CreatePR(branchName, "gitops", gitHubUser, base, title, body)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		log.Info().Msg(`waiting "atlantis plan" to start...`)
		time.Sleep(5 * time.Second)

		ok, err := gitHubClient.RetrySearchPullRequestComment(
			githubOwner,
			pkg.KubefirstGitOpsRepository,
			pullRequest,
			"To **apply** all unapplied plans from this pull request, comment",
			`waiting "atlantis plan" finish to proceed...`,
		)
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		if !ok {
			log.Info().Msg(`unable to run "atlantis plan"`)
			wg.Done()
			return
		}

		if err := gitHubClient.CommentPR(pullRequest, gitHubUser, "atlantis apply"); err != nil {
			log.Error().Err(err).Msg("")
		}
		wg.Done()
	}()

	_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/ingressroute.yaml", config.K1FolderPath))

	if err != nil {
		log.Error().Err(err).Msgf("failed to create ingress route to argocd: %s", err)
	}

	_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "delete", "ingress", "argocd-server")

	if err != nil {
		log.Error().Err(err).Msgf("failed to delete argocd primary ingress route: %s", err)
	}

	log.Info().Msg("Kubefirst installation almost finished successfully, please wait final setups steps")
	pkg.InformUser("Kubefirst installation almost finished successfully, please wait final setups steps", silentMode)

	// waiting GitHub/atlantis step
	wg.Wait()

	log.Info().Msg("sending mgmt cluster install completed metric")
	if useTelemetry {
		if err = wrappers.SendSegmentIoTelemetry("", pkg.MetricMgmtClusterInstallCompleted, cloud, gitProvider); err != nil {
			log.Error().Err(err).Msg("")
		}
		log.Info().Msg("Telemetry info sent")
	} else {
		pkg.InformUser("Telemetry skipped by user request", silentMode)
	}

	pkg.InformUser("Kubefirst installation finished successfully", silentMode)
	return nil

}
