package local

import (
	"context"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"log"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/metaphor"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
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
	gitOpsBranch   string
	gitOpsRepo     string
	metaphorBranch string
	adminEmail     string
	templateTag    string
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

	localCmd.Flags().BoolVar(&useTelemetry, "use-telemetry", true, "installer will not send telemetry about this installation")
	localCmd.Flags().BoolVar(&dryRun, "dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	localCmd.Flags().BoolVar(&silentMode, "silent", false, "enable silentMode mode will make the UI return less content to the screen")
	localCmd.Flags().BoolVar(&enableConsole, "enable-console", true, "If hand-off screen will be presented on a browser UI")

	// todo: get it from GH token , use it for console
	localCmd.Flags().StringVar(&adminEmail, "admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	localCmd.Flags().StringVar(&metaphorBranch, "metaphor-branch", "main", "metaphor application branch")
	localCmd.Flags().StringVar(&gitOpsBranch, "gitops-branch", "main", "version/branch used on git clone")
	localCmd.Flags().StringVar(&gitOpsRepo, "gitops-repo", "gitops", "")
	localCmd.Flags().StringVar(&templateTag, "template-tag", "",
		"when running a built version, and ldflag is set for the Kubefirst version, it will use this tag value to clone the templates (gitops and metaphor's)",
	)

	// on error, doesnt show helper/usage
	localCmd.SilenceUsage = true

	// wire up new commands
	localCmd.AddCommand(NewCommandConnect())

	return localCmd
}

func runLocal(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()

	//tools.RunInfo(cmd, args)

	progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
	progressPrinter.AddTracker("step-base", "Setup base cluster", 2)
	progressPrinter.AddTracker("step-apps", "Install apps to cluster", 4)

	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), silentMode)

	if useTelemetry {
		progressPrinter.AddTracker("step-telemetry", pkg.SendTelemetry, 2)
		if err := wrappers.SendSegmentIoTelemetry("", pkg.MetricMgmtClusterInstallStarted); err != nil {
			log.Println(err)
		}
		progressPrinter.IncrementTracker("step-telemetry", 1)
	}

	// todo need to add go channel to control when ngrok should close
	// and use context to handle closing the open goroutine/connection
	go pkg.RunNgrok(context.TODO(), pkg.LocalAtlantisURL)
	time.Sleep(5 * time.Second)

	if !viper.GetBool("kubefirst.done") {
		if viper.GetString("gitprovider") == "github" {
			log.Println("Installing Github version of Kubefirst")
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
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint)

		pkg.InformUser(fmt.Sprintf("Created gitops Repo in github.com/%s", viper.GetString("github.owner")), silentMode)
		progressPrinter.IncrementTracker("step-github", 1)
	} else {
		log.Println("already created github terraform resources")
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
		log.Println("already hydrated the github gitops repository")
	}
	progressPrinter.IncrementTracker("step-github", 1)

	// create kubernetes cluster
	executionControl = viper.GetBool("k3d.created")
	if !executionControl {
		pkg.InformUser("Creating K8S Cluster", silentMode)
		err := k3d.CreateK3dCluster()
		if err != nil {
			log.Println("Error installing k3d cluster")
			return err
		}
	} else {
		log.Println("already created k3d cluster")
	}
	progressPrinter.IncrementTracker("step-base", 1)
	progressPrinter.IncrementTracker("step-github", 1)

	// add secrets to cluster
	// todo there is a secret condition in AddK3DSecrets to this not checked
	executionControl = viper.GetBool("kubernetes.vault.secret.created")
	if !executionControl {
		err := k3d.AddK3DSecrets(dryRun)
		if err != nil {
			log.Println("Error AddK3DSecrets")
			return err
		}
	} else {
		log.Println("already added secrets to k3d cluster")
	}

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

		err := argocd.CreateInitialArgoCDRepository(config, argoCDConfig)
		if err != nil {
			log.Println("Error CreateInitialArgoCDRepository")
			return err
		}
	} else {
		log.Println("already created initial argocd repository")
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
		helm.AddRepoAndUpdateRepo(dryRun, helmRepo)
	}

	// helm install argocd
	executionControl = viper.GetBool("argocd.helm.install.complete")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), silentMode)
		helm.Install(dryRun, helmRepo)
	}
	progressPrinter.IncrementTracker("step-apps", 1)

	// argocd pods are running
	executionControl = viper.GetBool("argocd.ready")
	if !executionControl {
		argocd.WaitArgoCDToBeReady(dryRun)
		pkg.InformUser("ArgoCD is running, continuing", silentMode)
	} else {
		log.Println("already waited for argocd to be ready")
	}

	// ArgoCD port-forward
	argoCDStopChannel := make(chan struct{}, 1)
	defer func() {
		close(argoCDStopChannel)
	}()
	k8s.OpenPortForwardWrapper(
		pkg.ArgoCDPodName,
		pkg.ArgoCDNamespace,
		pkg.ArgoCDPodPort,
		pkg.ArgoCDPodLocalPort,
		argoCDStopChannel,
	)
	pkg.InformUser(fmt.Sprintf("port-forward to argocd is available at %s", viper.GetString("argocd.local.service")), silentMode)

	// argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("argocd.credentials.set")
	if !executionControl {
		pkg.InformUser("Setting argocd username and password credentials", silentMode)
		k8s.SetArgocdCreds(dryRun)
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
		err := argocd.ApplyRegistryLocal(dryRun)
		if err != nil {
			log.Println("Error applying registry application to argocd")
			return err
		}
	}

	progressPrinter.IncrementTracker("step-apps", 1)

	// vault in running state
	executionControl = viper.GetBool("vault.status.running")
	if !executionControl {
		pkg.InformUser("Waiting for vault to be ready", silentMode)
		vault.WaitVaultToBeRunning(dryRun)
	}

	// Vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardWrapper(
		pkg.VaultPodName,
		pkg.VaultNamespace,
		pkg.VaultPodPort,
		pkg.VaultPodLocalPort,
		vaultStopChannel,
	)

	k8s.LoopUntilPodIsReady(dryRun)

	minioStopChannel := make(chan struct{}, 1)
	defer func() {
		close(minioStopChannel)
	}()
	k8s.OpenPortForwardWrapper(
		pkg.MinioPodName,
		pkg.MinioNamespace,
		pkg.MinioPodPort,
		pkg.MinioPodLocalPort,
		minioStopChannel,
	)

	// todo: can I remove it?
	time.Sleep(20 * time.Second)

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
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint)

		pkg.InformUser("vault terraform executed successfully", silentMode)

		//* create vault configurerd secret
		// todo remove this code
		log.Println("creating vault configured secret")
		k8s.CreateVaultConfiguredSecret(dryRun, config)
		pkg.InformUser("Vault secret created", silentMode)
	} else {
		log.Println("already executed vault terraform")
	}

	// create users
	executionControl = viper.GetBool("terraform.users.apply.complete")
	if !executionControl {
		pkg.InformUser("applying users terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/users"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint)

		pkg.InformUser("executed users terraform successfully", silentMode)
		// progressPrinter.IncrementTracker("step-users", 1)
	} else {
		log.Println("already created users with terraform")
	}

	// TODO: K3D =>  NEED TO REMOVE local-backend.tf and rename remote-backend.md

	pkg.InformUser("Welcome to local kubefirst experience", silentMode)
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
		// Chartmuseum port-forward
		chartmuseumStopChannel := make(chan struct{}, 1)
		defer func() {
			close(chartmuseumStopChannel)
		}()
		k8s.OpenPortForwardWrapper(
			pkg.ChartmuseumPodName,
			pkg.ChartmuseumNamespace,
			pkg.ChartmuseumPodPort,
			pkg.ChartmuseumPodLocalPort,
			chartmuseumStopChannel,
		)

		pkg.AwaitHostNTimes("http://localhost:8181/health", 5, 5)
		viper.Set("chartmuseum.host.resolved", true)
		viper.WriteConfig()
	} else {
		log.Println("already resolved host for chartmuseum, continuing")
	}

	pkg.InformUser("Deploying metaphor applications", silentMode)
	err := metaphor.DeployMetaphorGithubLocal(dryRun, githubOwner, metaphorBranch, "")
	if err != nil {
		pkg.InformUser("Error deploy metaphor applications", silentMode)
		log.Println("Error running deployMetaphorCmd")
		log.Println(err)
	}

	// update terraform s3 backend to internal k8s dns (s3/minio bucket)
	err = pkg.UpdateTerraformS3BackendForK8sAddress()
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
		localRepo,
		remoteName,
		branchNameRef,
	)
	if err != nil {
		log.Println(err)
	}

	log.Println("sleeping after git commit with Minio backend update for Terraform")
	time.Sleep(3 * time.Second)

	// create a PR, atlantis will identify it's a Terraform change/file update and trigger atlantis plan
	// it's a goroutine since it can run in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// Atlantis port-forward
		atlantisStopChannel := make(chan struct{}, 1)
		defer func() {
			close(atlantisStopChannel)
		}()
		k8s.OpenPortForwardWrapper(
			pkg.AtlantisPodName,
			pkg.AtlantisNamespace,
			pkg.AtlantisPodPort,
			pkg.AtlantisPodLocalPort,
			atlantisStopChannel,
		)

		gitHubClient := githubWrapper.New()
		err = gitHubClient.CreatePR(branchName)
		if err != nil {
			fmt.Println(err)
		}
		log.Println(`waiting "atlantis plan" to start...`)
		time.Sleep(5 * time.Second)

		ok, err := gitHubClient.RetrySearchPullRequestComment(
			githubOwner,
			gitOpsRepo,
			"To **apply** all unapplied plans from this pull request, comment",
			`waiting "atlantis plan" finish to proceed...`,
		)
		if err != nil {
			log.Println(err)
		}

		if !ok {
			log.Println(`unable to run "atlantis plan"`)
			wg.Done()
			return
		}

		err = gitHubClient.CommentPR(1, "atlantis apply")
		if err != nil {
		}
		wg.Done()
	}()

	log.Println("sending mgmt cluster install completed metric")

	if useTelemetry {
		if err = wrappers.SendSegmentIoTelemetry("", pkg.MetricMgmtClusterInstallCompleted); err != nil {
			log.Println(err)
		}
		progressPrinter.IncrementTracker("step-telemetry", 1)
	}

	log.Println("Kubefirst installation finished successfully")
	pkg.InformUser("Kubefirst installation finished successfully", silentMode)

	// waiting GitHub/atlantis step
	wg.Wait()

	return nil

}
