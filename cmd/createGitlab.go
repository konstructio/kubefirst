/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/softserve"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createGitlabCmd represents the createGitlab command
var createGitlabCmd = &cobra.Command{
	Use:   "create-gitlab",
	Short: "create a kubefirst management cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("createGitlab called")
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

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		infoCmd.Run(cmd, args)
		progressPrinter.IncrementTracker("step-0", 1)

		progressPrinter.AddTracker("step-softserve", "Prepare Temporary Repo ", 4)
		progressPrinter.IncrementTracker("step-softserve", 1)
		if !globalFlags.UseTelemetry {
			informUser("Telemetry Disabled")
		}
		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		informUser("Creating K8S Cluster")
		terraform.ApplyBaseTerraform(globalFlags.DryRun, directory)
		progressPrinter.IncrementTracker("step-softserve", 1)

		restoreSSLCmd.Run(cmd, args)

		clientset, err := k8s.GetClientSet()
		if err != nil {
			panic(err.Error())
		}

		//! soft-serve was just applied

		softserve.CreateSoftServe(globalFlags.DryRun, config.KubeConfigPath)
		informUser("Created Softserve")
		progressPrinter.IncrementTracker("step-softserve", 1)
		informUser("Waiting Softserve")
		waitForNamespaceandPods(globalFlags.DryRun, config, "soft-serve", "app=soft-serve")
		progressPrinter.IncrementTracker("step-softserve", 1)
		// todo this should be replaced with something more intelligent
		log.Println("Waiting for soft-serve installation to complete...")
		if !globalFlags.DryRun {
			kPortForwardSoftServe, err := k8s.K8sPortForward(globalFlags.DryRun, "soft-serve", "svc/soft-serve", "8022:22")
			defer kPortForwardSoftServe.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error creating port-forward")
				return err
			}
			time.Sleep(20 * time.Second)
		}

		informUser("Softserve Update")
		softserve.ConfigureSoftServeAndPush(globalFlags.DryRun)
		progressPrinter.IncrementTracker("step-softserve", 1)

		progressPrinter.AddTracker("step-argo", "Deploy CI/CD ", 5)
		informUser("Deploy ArgoCD")
		progressPrinter.IncrementTracker("step-argo", 1)
		helm.InstallArgocd(globalFlags.DryRun)

		//! argocd was just helm installed
		waitArgoCDToBeReady(globalFlags.DryRun)

		informUser("ArgoCD Ready")
		progressPrinter.IncrementTracker("step-argo", 1)

		if !globalFlags.DryRun {
			kPortForwardArgocd, err = k8s.K8sPortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
			defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error creating port-forward")
				return err
			}

		}

		// log.Println("sleeping for 45 seconds, hurry up jared")
		// time.Sleep(45 * time.Second)

		informUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")))
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Setting argocd credentials")
		setArgocdCreds(globalFlags.DryRun)
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Getting an argocd auth token")

		progressPrinter.IncrementTracker("step-argo", 1)
		if !globalFlags.DryRun {
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
		waitVaultToBeRunning(globalFlags.DryRun)
		progressPrinter.IncrementTracker("step-gitlab", 1)
		if !globalFlags.DryRun {
			kPortForwardVault, err := k8s.K8sPortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
			defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error creating port-forward")
				return err
			}

		}
		loopUntilPodIsReady(globalFlags.DryRun)
		initializeVaultAndAutoUnseal(globalFlags.DryRun)
		informUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")))
		progressPrinter.IncrementTracker("step-gitlab", 1)

		informUser("Waiting gitlab to be ready")
		waitGitlabToBeReady(globalFlags.DryRun)
		log.Println("waiting for gitlab")
		waitForGitlab(globalFlags.DryRun, config)
		log.Println("gitlab is ready!")
		progressPrinter.IncrementTracker("step-gitlab", 1)

		if !globalFlags.DryRun {
			kPortForwardGitlab, err := k8s.K8sPortForward(globalFlags.DryRun, "gitlab", "svc/gitlab-webservice-default", "8888:8080")
			defer kPortForwardGitlab.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error creating port-forward")
				return err
			}
		}
		informUser(fmt.Sprintf("Gitlab available at %s", viper.GetString("gitlab.local.service")))
		progressPrinter.IncrementTracker("step-gitlab", 1)

		if !skipGitlab {
			// TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			informUser("Gitlab setup tokens")
			gitlab.ProduceGitlabTokens(globalFlags.DryRun)
			progressPrinter.IncrementTracker("step-gitlab", 1)
			informUser("Gitlab terraform")
			gitlab.ApplyGitlabTerraform(globalFlags.DryRun, directory)
			gitlab.GitlabKeyUpload(globalFlags.DryRun)
			informUser("Gitlab ready")
			progressPrinter.IncrementTracker("step-gitlab", 1)
		}
		if !skipVault {

			progressPrinter.AddTracker("step-vault", "Configure Vault", 2)
			informUser("waiting for vault unseal")

			log.Println("configuring vault")
			vault.ConfigureVault(globalFlags.DryRun, true)
			informUser("Vault configured")
			progressPrinter.IncrementTracker("step-vault", 1)

			log.Println("creating vault configured secret")
			createVaultConfiguredSecret(globalFlags.DryRun, config)
			informUser("Vault  secret created")
			progressPrinter.IncrementTracker("step-vault", 1)
		}
		progressPrinter.AddTracker("step-post-gitlab", "Finalize Gitlab updates", 5)
		if !viper.GetBool("gitlab.oidc-created") {
			vault.AddGitlabOidcApplications(globalFlags.DryRun)
			informUser("Added Gitlab OIDC")
			progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Waiting for Gitlab dns to propagate before continuing")
			gitlab.AwaitHost("gitlab", globalFlags.DryRun)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Pushing gitops repo to origin gitlab")
			// refactor: sounds like a new functions, should PushGitOpsToGitLab be renamed/update signature?
			viper.Set("gitlab.oidc-created", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.gitops-pushed") {
			gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "gitops") // todo: need to handle if this was already pushed, errors on failure)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo: keep one of the two git push functions, they're similar, but not exactly the same
			//gitlab.PushGitOpsToGitLab(globalFlags.DryRun)
			viper.Set("gitlab.gitops-pushed", true)
			viper.WriteConfig()
		}
		if !globalFlags.DryRun && !viper.GetBool("argocd.oidc-patched") {
			argocdSecretClient = clientset.CoreV1().Secrets("argocd")
			patchSecret(argocdSecretClient, "argocd-secret", "oidc.gitlab.clientSecret", viper.GetString("gitlab.oidc.argocd.secret"))

			argocdPodClient := clientset.CoreV1().Pods("argocd")
			k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			viper.Set("argocd.oidc-patched", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.metaphor-pushed") {
			informUser("Pushing metaphor repo to origin gitlab")
			gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor")
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo: keep one of the two git push functions, they're similar, but not exactly the same
			//gitlab.PushGitOpsToGitLab(globalFlags.DryRun)
			viper.Set("gitlab.metaphor-pushed", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.registered") {
			// informUser("Getting ArgoCD auth token")
			// token := argocd.GetArgocdAuthToken(globalFlags.DryRun)
			// progressPrinter.IncrementTracker("step-post-gitlab", 1)

			// informUser("Detaching the registry application from softserve")
			// argocd.DeleteArgocdApplicationNoCascade(globalFlags.DryRun, "registry", token)
			// progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Adding the registry application registered against gitlab")
			gitlab.ChangeRegistryToGitLab(globalFlags.DryRun)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo triage / force apply the contents adjusting
			// todo kind: Application .repoURL:

			// informUser("Waiting for argocd host to resolve")
			// gitlab.AwaitHost("argocd", globalFlags.DryRun)
			if !globalFlags.DryRun {
				argocdPodClient := clientset.CoreV1().Pods("argocd")
				kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				informUser("deleting argocd-server pod")
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
			informUser("waiting for argocd to be ready")
			waitArgoCDToBeReady(globalFlags.DryRun)

			informUser("Port forwarding to new argocd-server pod")
			if !globalFlags.DryRun {
				time.Sleep(time.Second * 20)
				kPortForwardArgocd, err = k8s.K8sPortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
				defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					log.Println("Error creating port-forward")
					return err
				}
				log.Println("sleeping for 40 seconds")
				time.Sleep(40 * time.Second)
			}

			informUser("Syncing the registry application")
			token := argocd.GetArgocdAuthToken(globalFlags.DryRun)

			if globalFlags.DryRun {
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
			argoCDHostReady := gitlab.AwaitHostNTimes("argocd", globalFlags.DryRun, 20)
			if argoCDHostReady {
				informUser("ArgoCD DNS is ready")
				break
			} else {
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
		}

		//!--

		if !skipVault {
			progressPrinter.AddTracker("step-vault-be", "Configure Vault Backend", 1)
			log.Println("configuring vault backend")
			vault.ConfigureVault(globalFlags.DryRun, false)
			informUser("Vault backend configured")
			progressPrinter.IncrementTracker("step-vault-be", 1)
		}
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createGitlabCmd)
	currentCommand := createGitlabCmd
	// todo: make this an optional switch and check for it or viper
	currentCommand.Flags().Bool("destroy", false, "destroy resources")
	currentCommand.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	currentCommand.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
	flagset.DefineGlobalFlags(currentCommand)
}
