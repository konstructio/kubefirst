package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var (
	useTelemetry   bool
	dryRun         bool
	silentMode     bool
	gitHubHost     string
	gitHubOwner    string
	gitHubUser     string
	gitOpsBranch   string
	gitOpsRepo     string
	cloud          string
	clusterName    string
	awsNodeSpot    bool // todo: add
	awsAssumeRole  string
	awsHostedZone  string
	metaphorBranch string
	gitProvider    string
	adminEmail     string
)

func NewCommand() *cobra.Command {

	localCmd := &cobra.Command{
		Use:   "local",
		Short: "Kubefirst localhost installation",
		Long:  "Kubefirst localhost enable a localhost installation without the requirement of a cloud provider.",
		RunE:  runLocal,
	}

	localCmd.Flags().BoolVar(&useTelemetry, "use-telemetry", true, "xinstaller will not send telemetry about this installation")
	localCmd.Flags().BoolVar(&dryRun, "dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	localCmd.Flags().BoolVar(&silentMode, "silent", false, "enable silentMode mode will make the UI return less content to the screen")
	localCmd.Flags().StringVar(&gitHubHost, "github-host", "github.com", "Github URL")
	localCmd.Flags().StringVar(&gitHubOwner, "github-owner", "", "Github owner of repos")
	localCmd.Flags().StringVar(&gitHubUser, "github-user", "", "Github user")
	localCmd.Flags().StringVar(&clusterName, "cluster-name", "kubefirst", "the cluster name, used to identify resources on cloud provider")
	localCmd.Flags().StringVar(&awsAssumeRole, "aws-assume-role", "", "instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html")
	localCmd.Flags().StringVar(&awsHostedZone, "hosted-zone-name", "", "the domain to provision the kubefirst platform in")
	localCmd.Flags().StringVar(&gitProvider, "git-provider", "github", "specify \"github\" or \"gitlab\" git provider. defaults to github.")
	localCmd.Flags().StringVar(&adminEmail, "admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")

	//initCmd.Flags().BoolVar(&awsNodeSpot, "aws-nodes-spot", false, "nodes spot on AWS EKS compute nodes")
	//initCmd.Flags().StringVar("s3-suffix", "", "unique identifier for s3 buckets")
	//initCmd.Flags().String("profile", "", "AWS profile located at ~/.aws/config")
	//initCmd.Flags().String("region", "", "the region to provision the cloud resources in")

	localCmd.Flags().StringVar(&metaphorBranch, "metaphor-branch", "main", "metaphro application branch")
	localCmd.Flags().StringVar(&gitOpsBranch, "gitops-branch", "main", "version/branch used on git clone - former: version-gitops flag")
	localCmd.Flags().StringVar(&gitOpsRepo, "gitops-repo", "gitops", "")
	//initCmd.Flags().StringP("config", "c", "", "File to be imported to bootstrap configs")
	//viper.BindPFlag("config.file", currentCommand.Flags().Lookup("config-load"))

	return localCmd
}

func runLocal(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()

	// todo: viper struct
	gitopsRepo, err := cmd.Flags().GetString("gitops-repo")
	if err != nil {
		return err
	}
	gitopsOwner, err := cmd.Flags().GetString("gitops-repo")
	if err != nil {
		return err
	}
	err = NewInit(gitopsRepo, gitopsOwner, "github", "main")
	if err != nil {
		log.Println(err)
		return err
	}

	// telemetry
	if useTelemetry {
		// Instantiates a SegmentIO client to send messages to the segment API.
		segmentIOClientStart := analytics.New(pkg.SegmentIOWriteKey)

		// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
		// to force flush in case there is still some pending message in the SegmentIO library queue.
		defer func(segmentIOClient analytics.Client) {
			err := segmentIOClient.Close()
			if err != nil {
				log.Println(err)
			}
		}(segmentIOClientStart)

		telemetryDomainStart, err := domain.NewTelemetry(
			pkg.MetricMgmtClusterInstallStarted,
			"",
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		telemetryServiceStart := services.NewSegmentIoService(segmentIOClientStart)
		telemetryHandlerStart := handlers.NewTelemetryHandler(telemetryServiceStart)

		err = telemetryHandlerStart.SendCountMetric(telemetryDomainStart)
		if err != nil {
			log.Println(err)
		}
	}

	// todo need to add go channel to control when ngrok should close
	// and use context to handle closing the open goroutine/connection
	go pkg.RunNgrok(context.TODO(), pkg.LocalAtlantisURL)
	time.Sleep(5 * time.Second)

	if !viper.GetBool("kubefirst.done") {
		if viper.GetString("gitprovider") == "github" {
			log.Println("Installing Github version of Kubefirst")
			viper.Set("git.mode", "github")
			// if not local it is AWS for now
			// todo: internal
			err = k3d.CreateK3dCluster()
			if err != nil {
				return err
			}
		}
		viper.Set("kubefirst.done", true)
		viper.WriteConfig()
	}
	// start

	//infoCmd need to be before the bars or it is printed in between bars:
	//Let's try to not move it on refactors
	//infoCmd.Run(cmd, args)
	var kPortForwardArgocd *exec.Cmd
	progressPrinter.AddTracker("step-0", "Process Parameters", 1)
	progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
	progressPrinter.AddTracker("step-base", "Setup base cluster", 2)
	progressPrinter.AddTracker("step-apps", "Install apps to cluster", 5)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), silentMode)

	progressPrinter.IncrementTracker("step-0", 1)

	if !useTelemetry {
		pkg.InformUser("Telemetry Disabled", silentMode)
	}

	executionControl := viper.GetBool("terraform.github.apply.complete")
	//* create github teams in the org and gitops repo
	if !executionControl {
		pkg.InformUser("Creating github resources with terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint)

		pkg.InformUser(fmt.Sprintf("Created gitops Repo in github.com/%s", viper.GetString("github.owner")), silentMode)
		progressPrinter.IncrementTracker("step-github", 1)
	} else {
		log.Println("already created github terraform resources")
	}

	//* push our locally detokenized gitops repo to remote github
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

	//* create kubernetes cluster
	executionControl = viper.GetBool("k3d.created")
	if !executionControl {
		pkg.InformUser("Creating K8S Cluster", silentMode)
		err = k3d.CreateK3dCluster()
		if err != nil {
			log.Println("Error installing k3d cluster")
			return err
		}
		progressPrinter.IncrementTracker("step-base", 1)
	} else {
		log.Println("already created k3d cluster")
	}
	progressPrinter.IncrementTracker("step-github", 1)

	// add secrets to cluster
	// todo there is a secret condition in AddK3DSecrets to this not checked
	executionControl = viper.GetBool("kubernetes.vault.secret.created")
	if !executionControl {
		err = k3d.AddK3DSecrets(dryRun)
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
		//Enterprise users need to be able to set the hostname for git.
		gitopsRepo := fmt.Sprintf("git@%s:%s/gitops.git", viper.GetString("github.host"), viper.GetString("github.owner"))
		err = argocd.CreateInitialArgoCDRepository(gitopsRepo)
		if err != nil {
			log.Println("Error CreateInitialArgoCDRepository")
			return err
		}
	} else {
		log.Println("already created initial argocd repository")
	}

	//* helm add argo repository && update
	helmRepo := helm.HelmRepo{}
	helmRepo.RepoName = "argo"
	helmRepo.RepoURL = "https://argoproj.github.io/argo-helm"
	helmRepo.ChartName = "argo-cd"
	helmRepo.Namespace = "argocd"
	helmRepo.ChartVersion = "4.10.5"

	executionControl = viper.GetBool("argocd.helm.repo.updated")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL), silentMode)
		helm.AddRepoAndUpdateRepo(dryRun, helmRepo)
	}

	//* helm install argocd
	executionControl = viper.GetBool("argocd.helm.install.complete")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), silentMode)
		helm.Install(dryRun, helmRepo)
	}
	progressPrinter.IncrementTracker("step-apps", 1)

	//* argocd pods are running
	executionControl = viper.GetBool("argocd.ready")
	if !executionControl {
		argocd.WaitArgoCDToBeReady(dryRun)
		pkg.InformUser("ArgoCD is running, continuing", silentMode)
	} else {
		log.Println("already waited for argocd to be ready")
	}

	//* establish port-forward
	kPortForwardArgocd, err = k8s.PortForward(dryRun, "argocd", "svc/argocd-server", "8080:80")
	defer func() {
		err = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Println("Error closing kPortForwardArgocd")
		}
	}()
	pkg.InformUser(fmt.Sprintf("port-forward to argocd is available at %s", viper.GetString("argocd.local.service")), silentMode)

	//* argocd pods are ready, get and set credentials
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

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("argocd.registry.applied")
	if !executionControl {
		pkg.InformUser("applying the registry application to argocd", silentMode)
		err = argocd.ApplyRegistryLocal(dryRun)
		if err != nil {
			log.Println("Error applying registry application to argocd")
			return err
		}
	}

	progressPrinter.IncrementTracker("step-apps", 1)

	//* vault in running state
	executionControl = viper.GetBool("vault.status.running")
	if !executionControl {
		pkg.InformUser("Waiting for vault to be ready", silentMode)
		vault.WaitVaultToBeRunning(dryRun)
		if err != nil {
			log.Println("error waiting for vault to become running")
			return err
		}
	}
	kPortForwardVault, err := k8s.PortForward(dryRun, "vault", "svc/vault", "8200:8200")
	defer func() {
		err = kPortForwardVault.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Println("Error closing kPortForwardVault")
		}
	}()

	k8s.LoopUntilPodIsReady(dryRun)
	kPortForwardMinio, err := k8s.PortForward(dryRun, "minio", "svc/minio", "9000:9000")
	defer func() {
		err = kPortForwardMinio.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Println("Error closing kPortForwardMinio")
		}
	}()

	//* configure vault with terraform
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

	//* create users
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
	time.Sleep(1 * time.Second)
	progressPrinter.IncrementTracker("step-apps", 1)
	progressPrinter.IncrementTracker("step-base", 1)
	progressPrinter.IncrementTracker("step-apps", 1)

	// end

	if !viper.GetBool("chartmuseum.host.resolved") {

		//* establish port-forward
		var kPortForwardChartMuseum *exec.Cmd
		kPortForwardChartMuseum, err = k8s.PortForward(dryRun, "chartmuseum", "svc/chartmuseum", "8181:8080")
		defer func() {
			err = kPortForwardChartMuseum.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardChartMuseum")
			}
		}()
		pkg.AwaitHostNTimes("http://localhost:8181/health", 5, 5)
		viper.Set("chartmuseum.host.resolved", true)
		viper.WriteConfig()
	} else {
		log.Println("already resolved host for chartmuseum, continuing")
	}

	// todo: uncomment it
	//pkg.InformUser("Deploying metaphor applications", silentMode)
	//err = deployMetaphorCmd.RunE(cmd, args)
	//if err != nil {
	//	pkg.InformUser("Error deploy metaphor applications", silentMode)
	//	log.Println("Error running deployMetaphorCmd")
	//	return err
	//}

	//kPortForwardAtlantis, err := k8s.PortForward(dryRun, "atlantis", "svc/atlantis", "4141:80")
	//defer func() {
	//	err = kPortForwardAtlantis.Process.Signal(syscall.SIGTERM)
	//	if err != nil {
	//		log.Println("error closing kPortForwardAtlantis")
	//	}
	//}()

	// ---

	// update terraform s3 backend to internal k8s dns (s3/minio bucket)
	err = pkg.ReplaceS3Backend()
	if err != nil {
		return err
	}

	// create a new branch and push changes
	githubHost = viper.GetString("github.host")
	githubOwner = viper.GetString("github.owner")
	remoteName = "github"
	localRepo = "gitops"
	branchName := "update-s3-backend"
	branchNameRef := plumbing.ReferenceName("refs/heads/" + branchName)

	gitClient.UpdateLocalTFFilesAndPush(
		githubHost,
		githubOwner,
		localRepo,
		remoteName,
		branchNameRef,
	)

	fmt.Println("sleeping after commit...")
	time.Sleep(3 * time.Second)

	// create a PR, atlantis will identify it's a terraform change/file update and,
	// trigger atlantis plan
	g := githubWrapper.New()
	err = g.CreatePR(branchName)
	if err != nil {
		fmt.Println(err)
	}
	log.Println("sleeping after create PR...")
	time.Sleep(5 * time.Second)
	log.Println("sleeping... atlantis plan should be running")
	time.Sleep(5 * time.Second)

	fmt.Println("sleeping before apply...")
	time.Sleep(120 * time.Second)

	// after 120 seconds, it will comment in the PR with atlantis plan
	err = g.CommentPR(1, "atlantis apply")
	if err != nil {
		fmt.Println(err)
	}

	log.Println("sending mgmt cluster install completed metric")

	if useTelemetry {
		// Instantiates a SegmentIO client to send messages to the segment API.
		segmentIOClientCompleted := analytics.New(pkg.SegmentIOWriteKey)

		// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
		// to force flush in case there is still some pending message in the SegmentIO library queue.
		defer func(segmentIOClientCompleted analytics.Client) {
			err := segmentIOClientCompleted.Close()
			if err != nil {
				log.Println(err)
			}
		}(segmentIOClientCompleted)

		telemetryDomainCompleted, err := domain.NewTelemetry(
			pkg.MetricMgmtClusterInstallCompleted,
			"",
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		telemetryServiceCompleted := services.NewSegmentIoService(segmentIOClientCompleted)
		telemetryHandlerCompleted := handlers.NewTelemetryHandler(telemetryServiceCompleted)

		err = telemetryHandlerCompleted.SendCountMetric(telemetryDomainCompleted)
		if err != nil {
			log.Println(err)
		}
	}

	log.Println("Kubefirst installation finished successfully")
	pkg.InformUser("Kubefirst installation finished successfully", silentMode)

	// todo: temporary code to enable console for localhost / enable it back!
	//err = postInstallCmd.RunE(cmd, args)
	//if err != nil {
	//	pkg.InformUser("Error starting apps from post-install", silentMode)
	//	log.Println("Error running postInstallCmd")
	//	return err
	//}

	return nil

}

func NewInit(gitopsRepo string, gitopsOwner string, gitProvider string, metaphorBranch string) error {

	//tools.RunInfo(cmd, args)
	// todo: load it from viper
	config := configs.ReadConfig()

	// set default values
	viper.Set("gitops.repo", gitOpsRepo)
	// todo: do we need it?
	viper.Set("gitops.owner", "kubefirst")

	viper.Set("gitprovider", gitProvider)
	viper.Set("metaphor.branch", metaphorBranch)
	viper.WriteConfig()
	//
	if err := pkg.ValidateK1Folder(config.K1FolderPath); err != nil {
		return err
	}

	// todo: wrap business logic into the handler
	if config.GitHubPersonalAccessToken == "" {

		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)
		gitHubAccessToken, err := gitHubHandler.AuthenticateUser()
		if err != nil {
			return err
		}

		if len(gitHubAccessToken) == 0 {
			return errors.New("unable to retrieve a GitHub token for the user")
		}

		viper.Set("github.token", gitHubAccessToken)
		err = viper.WriteConfig()
		if err != nil {
			return err
		}

		// todo: set common way to load env. values (viper->struct->load-env)
		// todo: use viper file to load it, not load env. value
		if err := os.Setenv("GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
			return err
		}
		log.Println("\nGITHUB_AUTH_TOKEN set via OAuth")
	}

	// review it
	viper.Set("gitops.branch", gitOpsBranch)
	viper.Set("github.owner", viper.GetString("github.user"))
	viper.Set("cloud", pkg.CloudK3d)
	viper.Set("cluster-name", clusterName)
	viper.Set("adminemail", adminEmail)
	viper.WriteConfig()

	if silentMode {
		pkg.InformUser(
			"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
			silentMode,
		)
	}

	progressPrinter.AddTracker("step-download", pkg.DownloadDependencies, 3)
	progressPrinter.AddTracker("step-gitops", pkg.CloneAndDetokenizeGitOpsTemplate, 1)
	progressPrinter.AddTracker("step-ssh", pkg.CreateSSHKey, 1)
	progressPrinter.AddTracker("step-telemetry", pkg.SendTelemetry, 1)

	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), silentMode)

	log.Println("sending init started metric")

	// todo:
	var telemetryHandler handlers.TelemetryHandler
	if useTelemetry {

		// Instantiates a SegmentIO client to use send messages to the segment API.
		segmentIOClient := analytics.New(pkg.SegmentIOWriteKey)

		// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
		// to force flush in case there is still some pending message in the SegmentIO library queue.
		defer func(segmentIOClient analytics.Client) {
			err := segmentIOClient.Close()
			if err != nil {
				log.Println(err)
			}
		}(segmentIOClient)

		// validate telemetryDomain data
		telemetryDomain, err := domain.NewTelemetry(
			pkg.MetricInitStarted,
			awsHostedZone,
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		telemetryService := services.NewSegmentIoService(segmentIOClient)
		telemetryHandler = handlers.NewTelemetryHandler(telemetryService)

		err = telemetryHandler.SendCountMetric(telemetryDomain)
		if err != nil {
			log.Println(err)
		}
	}

	// todo: set constants
	viper.Set("argocd.local.service", "http://localhost:8080")
	viper.Set("gitlab.local.service", "http://localhost:8888")
	viper.Set("vault.local.service", "http://localhost:8200")
	// used for letsencrypt notifications and the gitlab root account

	atlantisWebhookSecret := pkg.Random(20)
	viper.Set("github.atlantis.webhook.secret", atlantisWebhookSecret)

	viper.WriteConfig()

	//! tracker 0
	log.Println("installing kubefirst dependencies")
	progressPrinter.IncrementTracker("step-download", 1)
	err := downloadManager.DownloadTools(config)
	if err != nil {
		return err
	}
	log.Println("dependency installation complete")
	progressPrinter.IncrementTracker("step-download", 1)
	err = downloadManager.DownloadLocalTools(config)
	if err != nil {
		return err
	}

	//Fix incomplete bar, please don't remove it.
	progressPrinter.IncrementTracker("step-download", 1)

	//! tracker 5
	log.Println("creating an ssh key pair for your new cloud infrastructure")
	pkg.CreateSshKeyPair()
	log.Println("ssh key pair creation complete")
	progressPrinter.IncrementTracker("step-ssh", 1)

	//! tracker 6

	repo.PrepareKubefirstTemplateRepo(dryRun, config, viper.GetString("github.owner"), viper.GetString("gitops.repo"), viper.GetString("gitops.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of gitops-template repository complete")
	progressPrinter.IncrementTracker("step-gitops", 1)

	log.Println("sending init completed metric")

	if useTelemetry {
		telemetryInitCompleted, err := domain.NewTelemetry(
			pkg.MetricInitCompleted,
			awsHostedZone,
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		err = telemetryHandler.SendCountMetric(telemetryInitCompleted)
		if err != nil {
			log.Println(err)
		}
	}

	viper.WriteConfig()

	//! tracker 8
	progressPrinter.IncrementTracker("step-telemetry", 1)
	time.Sleep(time.Millisecond * 100)

	pkg.InformUser("init is done!\n", silentMode)

	return nil
}
