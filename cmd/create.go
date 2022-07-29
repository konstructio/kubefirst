package cmd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/softserve"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a kubefirst management cluster",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(4)

		var kPortForwardArgocd *exec.Cmd
		progressPrinter.AddTracker("step-0", "Process Parameters", 1)
		config := configs.ReadConfig()

		skipVault, err := cmd.Flags().GetBool("skip-vault")
		if err != nil {
			log.Panic(err)
		}
		skipGitlab, err := cmd.Flags().GetBool("skip-gitlab")
		if err != nil {
			log.Panic(err)
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Panic(err)
		}

		useTelemetry, err := cmd.Flags().GetBool("use-telemetry")
		if err != nil {
			log.Panic(err)
		}

		infoCmd.Run(cmd, args)
		progressPrinter.IncrementTracker("step-0", 1)

		progressPrinter.AddTracker("step-softserve", "Prepare Temporary Repo ", 4)
		sendStartedInstallTelemetry(dryRun, useTelemetry)
		progressPrinter.IncrementTracker("step-softserve", 1)
		if !useTelemetry {
			informUser("Telemetry Disabled")
		}
		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		informUser("Creating K8S Cluster")
		terraform.ApplyBaseTerraform(dryRun, directory)
		progressPrinter.IncrementTracker("step-softserve", 1)

		restoreSSLCmd.Run(cmd, args)

		kubeconfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
		if err != nil {
			panic(err.Error())
		}
		clientset, err := kubernetes.NewForConfig(kubeconfig)
		if err != nil {
			panic(err.Error())
		}

		//! soft-serve was just applied

		softserve.CreateSoftServe(dryRun, config.KubeConfigPath)
		informUser("Created Softserve")
		progressPrinter.IncrementTracker("step-softserve", 1)
		informUser("Waiting Softserve")
		waitForNamespaceandPods(dryRun, config, "soft-serve", "app=soft-serve")
		progressPrinter.IncrementTracker("step-softserve", 1)
		// todo this should be replaced with something more intelligent
		log.Println("Waiting for soft-serve installation to complete...")
		if !dryRun {
			var kPortForwardSoftServeOutb, kPortForwardSoftServeErrb bytes.Buffer
			time.Sleep(60 * time.Second)
			kPortForwardSoftServe := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
			kPortForwardSoftServe.Stdout = &kPortForwardSoftServeOutb
			kPortForwardSoftServe.Stderr = &kPortForwardSoftServeErrb
			err = kPortForwardSoftServe.Start()
			defer kPortForwardSoftServe.Process.Signal(syscall.SIGTERM)
			if err != nil {
				// If it doesn't error, we kinda don't care much.
				log.Printf("Commad Execution STDOUT: %s", kPortForwardSoftServeOutb.String())
				log.Printf("Commad Execution STDERR: %s", kPortForwardSoftServeErrb.String())
				log.Panicf("error: failed to port-forward to soft-serve %s", err)
			}
			time.Sleep(20 * time.Second)
		}

		informUser("Softserve Update")
		softserve.ConfigureSoftServeAndPush(dryRun)
		progressPrinter.IncrementTracker("step-softserve", 1)

		progressPrinter.AddTracker("step-argo", "Deploy CI/CD ", 5)
		informUser("Deploy ArgoCD")
		progressPrinter.IncrementTracker("step-argo", 1)
		helm.InstallArgocd(dryRun)

		//! argocd was just helm installed
		waitArgoCDToBeReady(dryRun)

		informUser("ArgoCD Ready")
		progressPrinter.IncrementTracker("step-argo", 1)

		if !dryRun {
			var kPortForwardArgocdOutb, kPortForwardArgocdErrb bytes.Buffer
			kPortForwardArgocd = exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
			kPortForwardArgocd.Stdout = &kPortForwardArgocdOutb
			kPortForwardArgocd.Stderr = &kPortForwardArgocdErrb
			err = kPortForwardArgocd.Start()
			defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Printf("Commad Execution STDOUT: %s", kPortForwardArgocdOutb.String())
				log.Printf("Commad Execution STDERR: %s", kPortForwardArgocdErrb.String())
				log.Panicf("error: failed to port-forward to argocd in main thread %s", err)
			}
		}

		// log.Println("sleeping for 45 seconds, hurry up jared")
		// time.Sleep(45 * time.Second)

		informUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")))
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Setting argocd credentials")
		setArgocdCreds(dryRun)
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Getting an argocd auth token")

		progressPrinter.IncrementTracker("step-argo", 1)
		if !dryRun {
			_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/helpers/registry.yaml", config.K1FolderPath))
			if err != nil {
				log.Panicf("failed to call execute kubectl apply of argocd patch to adopt gitlab: %s", err)
			}
			time.Sleep(45 * time.Second)
		}
		progressPrinter.IncrementTracker("step-argo", 1)

		//!
		//* we need to stop here and wait for the vault namespace to exist and the vault pod to be ready
		//!
		progressPrinter.AddTracker("step-gitlab", "Setup Gitlab", 6)
		informUser("Waiting vault to be ready")
		waitVaultToBeRunning(dryRun)
		progressPrinter.IncrementTracker("step-gitlab", 1)
		if !dryRun {
			var kPortForwardVaultOutb, kPortForwardVaultErrb bytes.Buffer
			kPortForwardVault := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
			kPortForwardVault.Stdout = &kPortForwardVaultOutb
			kPortForwardVault.Stderr = &kPortForwardVaultErrb
			err = kPortForwardVault.Start()
			defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
			if err != nil {
				// If it doesn't error, we kinda don't care much.
				log.Printf("Commad Execution STDOUT: %s", kPortForwardVaultOutb.String())
				log.Printf("Commad Execution STDERR: %s", kPortForwardVaultErrb.String())
				log.Panicf("error: failed to port-forward to vault in main thread %s", err)
			}
		}
		loopUntilPodIsReady(dryRun)
		initializeVaultAndAutoUnseal(dryRun)
		informUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")))
		progressPrinter.IncrementTracker("step-gitlab", 1)

		informUser("Waiting gitlab to be ready")
		waitGitlabToBeReady(dryRun)
		log.Println("waiting for gitlab")
		waitForGitlab(dryRun, config)
		log.Println("gitlab is ready!")
		progressPrinter.IncrementTracker("step-gitlab", 1)

		if !dryRun {
			var kPortForwardGitlabOutb, kPortForwardGitlabErrb bytes.Buffer
			kPortForwardGitlab := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
			kPortForwardGitlab.Stdout = &kPortForwardGitlabOutb
			kPortForwardGitlab.Stderr = &kPortForwardGitlabErrb
			err = kPortForwardGitlab.Start()
			defer kPortForwardGitlab.Process.Signal(syscall.SIGTERM)
			if err != nil {
				// If it doesn't error, we kinda don't care much.
				log.Printf("Commad Execution STDOUT: %s", kPortForwardGitlabOutb.String())
				log.Printf("Commad Execution STDERR: %s", kPortForwardGitlabErrb.String())
				log.Panicf("error: failed to port-forward to gitlab in main thread %s", err)
			}
		}
		informUser(fmt.Sprintf("Gitlab available at %s", viper.GetString("gitlab.local.service")))
		progressPrinter.IncrementTracker("step-gitlab", 1)

		if !skipGitlab {
			// TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			informUser("Gitlab setup tokens")
			gitlab.ProduceGitlabTokens(dryRun)
			progressPrinter.IncrementTracker("step-gitlab", 1)
			informUser("Gitlab terraform")
			gitlab.ApplyGitlabTerraform(dryRun, directory)
			gitlab.GitlabKeyUpload(dryRun)
			informUser("Gitlab ready")
			progressPrinter.IncrementTracker("step-gitlab", 1)
		}
		if !skipVault {

			progressPrinter.AddTracker("step-vault", "Configure Vault", 2)
			informUser("waiting for vault unseal")

			log.Println("configuring vault")
			vault.ConfigureVault(dryRun)
			informUser("Vault configured")
			progressPrinter.IncrementTracker("step-vault", 1)

			log.Println("creating vault configured secret")
			createVaultConfiguredSecret(dryRun, config)
			informUser("Vault  secret created")
			progressPrinter.IncrementTracker("step-vault", 1)
		}
		progressPrinter.AddTracker("step-post-gitlab", "Finalize Gitlab updates", 5)
		if !viper.GetBool("gitlab.oidc-created") {
			vault.AddGitlabOidcApplications(dryRun)
			informUser("Added Gitlab OIDC")
			progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Waiting for Gitlab dns to propagate before continuing")
			gitlab.AwaitHost("gitlab", dryRun)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Pushing gitops repo to origin gitlab")
			// refactor: sounds like a new functions, should PushGitOpsToGitLab be renamed/update signature?
			viper.Set("gitlab.oidc-created", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.gitops-pushed") {
			gitlab.PushGitRepo(dryRun, config, "gitlab", "gitops") // todo: need to handle if this was already pushed, errors on failure)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo: keep one of the two git push functions, they're similar, but not exactly the same
			//gitlab.PushGitOpsToGitLab(dryRun)
			viper.Set("gitlab.gitops-pushed", true)
			viper.WriteConfig()
		}
		if !dryRun && !viper.GetBool("argocd.oidc-patched") {
			argocdSecretClient = clientset.CoreV1().Secrets("argocd")
			patchSecret(argocdSecretClient, "argocd-secret", "oidc.gitlab.clientSecret", viper.GetString("gitlab.oidc.argocd.secret"))

			argocdPodClient := clientset.CoreV1().Pods("argocd")
			k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			viper.Set("argocd.oidc-patched", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.metaphor-pushed") {
			informUser("Pushing metaphor repo to origin gitlab")
			gitlab.PushGitRepo(dryRun, config, "gitlab", "metaphor")
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo: keep one of the two git push functions, they're similar, but not exactly the same
			//gitlab.PushGitOpsToGitLab(dryRun)
			viper.Set("gitlab.metaphor-pushed", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.registered") {
			// informUser("Getting ArgoCD auth token")
			// token := argocd.GetArgocdAuthToken(dryRun)
			// progressPrinter.IncrementTracker("step-post-gitlab", 1)

			// informUser("Detaching the registry application from softserve")
			// argocd.DeleteArgocdApplicationNoCascade(dryRun, "registry", token)
			// progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Adding the registry application registered against gitlab")
			gitlab.ChangeRegistryToGitLab(dryRun)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo triage / force apply the contents adjusting
			// todo kind: Application .repoURL:

			// informUser("Waiting for argocd host to resolve")
			// gitlab.AwaitHost("argocd", dryRun)
			if !dryRun {
				argocdPodClient := clientset.CoreV1().Pods("argocd")
				kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				informUser("deleting argocd-server pod")
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
			informUser("waiting for argocd to be ready")
			waitArgoCDToBeReady(dryRun)

			informUser("Port forwarding to new argocd-server pod")
			if !dryRun {
				time.Sleep(time.Second * 20)
				var kPortForwardArgocdOutb, kPortForwardArgocdErrb bytes.Buffer
				config := configs.ReadConfig()
				kPortForwardArgocd := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
				kPortForwardArgocd.Stdout = &kPortForwardArgocdOutb
				kPortForwardArgocd.Stderr = &kPortForwardArgocdErrb
				err = kPortForwardArgocd.Start()
				defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					log.Printf("Commad Execution STDOUT: %s", kPortForwardArgocdOutb.String())
					log.Printf("Commad Execution STDERR: %s", kPortForwardArgocdErrb.String())
					log.Panicf("error: failed to port-forward to argocd in main thread %s", err)
				}
				log.Println("sleeping for 40 seconds")
				time.Sleep(40 * time.Second)
			}

			informUser("Syncing the registry application")
			token := argocd.GetArgocdAuthToken(dryRun)

			if dryRun {
				log.Printf("[#99] Dry-run mode, Sync ArgoCD skipped")
			} else {
				// todo: create ArgoCD struct, and host dependencies (like http client)
				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				httpClient := http.Client{Transport: customTransport}

				// retry to sync ArgoCD application until reaches the maximum attempts
				argoCDIsReady, err := argocd.SyncRetry(&httpClient, 120, 5, "registry", token)
				if err != nil {
					log.Printf("something went wrong during ArgoCD sync step, error is: %v", err)
				}

				if !argoCDIsReady {
					log.Println("unable to sync ArgoCD application, continuing...")
				}
			}

			viper.Set("gitlab.registered", true)
			viper.WriteConfig()
		}

		//!--
		// Wait argocd cert to work, or force restart
		argocdPodClient := clientset.CoreV1().Pods("argocd")
		for i := 1; i < 15; i++ {
			argoCDHostReady := gitlab.AwaitHostNTimes("argocd", dryRun, 20)
			if argoCDHostReady {
				informUser("ArgoCD DNS is ready")
				break
			} else {
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
		}

		//!--

		sendCompleteInstallTelemetry(dryRun, useTelemetry)
		time.Sleep(time.Millisecond * 100)

		// prepare data for the handoff report
		clusterData := reports.CreateHandOff{
			AwsAccountId:      viper.GetString("aws.accountid"),
			AwsHostedZoneName: viper.GetString("aws.hostedzonename"),
			AwsRegion:         viper.GetString("aws.region"),
			ClusterName:       viper.GetString("cluster-name"),

			GitlabURL:      fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.hostedzonename")),
			GitlabUser:     "root",
			GitlabPassword: viper.GetString("gitlab.root.password"),

			RepoGitops:   fmt.Sprintf("https://gitlab.%s/kubefirst/gitops", viper.GetString("aws.hostedzonename")),
			RepoMetaphor: fmt.Sprintf("https://gitlab.%s/kubefirst/metaphor", viper.GetString("aws.hostedzonename")),

			VaultUrl:   fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename")),
			VaultToken: viper.GetString("vault.token"),

			ArgoCDUrl:      fmt.Sprintf("https://argocd.%s", viper.GetString("aws.hostedzonename")),
			ArgoCDUsername: viper.GetString("argocd.admin.username"),
			ArgoCDPassword: viper.GetString("argocd.admin.password"),

			ArgoWorkflowsUrl: fmt.Sprintf("https://argo.%s", viper.GetString("aws.hostedzonename")),
			AtlantisUrl:      fmt.Sprintf("https://atlantis.%s", viper.GetString("aws.hostedzonename")),
			ChartMuseumUrl:   fmt.Sprintf("https://chartmuseum.%s", viper.GetString("aws.hostedzonename")),

			MetaphorDevUrl:        fmt.Sprintf("https://metaphor-development.%s", viper.GetString("aws.hostedzonename")),
			MetaphorStageUrl:      fmt.Sprintf("https://metaphor-staging.%s", viper.GetString("aws.hostedzonename")),
			MetaphorProductionUrl: fmt.Sprintf("https://metaphor-production.%s", viper.GetString("aws.hostedzonename")),
		}

		// build the string that will be sent to the report
		handOffData := reports.BuildCreateHandOffReport(clusterData)
		// call handoff report and apply style
		reports.CommandSummary(handOffData)

	},
}

func init() {
	clusterCmd.AddCommand(createCmd)

	// todo: make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	createCmd.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	createCmd.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
	createCmd.Flags().Bool("use-telemetry", true, "installer will not send telemetry about this installation")

}
