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

	"github.com/kubefirst/kubefirst/internal/terraform"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createGithubK3dCmd represents the createGithub command
var createGithubK3dCmd = &cobra.Command{
	Use:   "create-github-k3d",
	Short: "create a kubefirst management cluster with github as Git Repo in k3d cluster",
	Long:  `Create a kubefirst cluster using github as the Git Repo and setup integrations`,
	RunE: func(cmd *cobra.Command, args []string) error {

		config := configs.ReadConfig()
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		//infoCmd need to be before the bars or it is printed in between bars:
		//Let's try to not move it on refactors
		infoCmd.Run(cmd, args)
		var kPortForwardArgocd *exec.Cmd
		progressPrinter.AddTracker("step-0", "Process Parameters", 1)
		progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
		progressPrinter.AddTracker("step-base", "Setup base cluster", 2)
		//progressPrinter.AddTracker("step-ecr", "Setup ECR/Docker Registries", 1) // todo remove this step, its baked into github repo
		progressPrinter.AddTracker("step-apps", "Install apps to cluster", 6)
		progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), globalFlags.SilentMode)

		progressPrinter.IncrementTracker("step-0", 1)

		if !globalFlags.UseTelemetry {
			informUser("Telemetry Disabled", globalFlags.SilentMode)
		}

		executionControl := viper.GetBool("terraform.github-k3d.apply.executed")
		//* create github teams in the org and gitops repo
		if !executionControl {
			informUser("Creating github resources with terraform", globalFlags.SilentMode)

			tfEntrypoint := config.GitOpsRepoPath + "/terraform/github-k3d"
			terraform.InitApplyAutoApprove(globalFlags.DryRun, tfEntrypoint)

			informUser(fmt.Sprintf("Created GitOps Repo in github.com/%s", viper.GetString("github.owner")), globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-github", 1)
		}

		//* push our locally detokenized gitops repo to remote github
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")
		localRepo := "gitops"
		remoteName := "github"
		executionControl = viper.GetBool("github.gitops.hydrated") // todo fix this executionControl value `github.detokenized-gitops.pushed`?
		if !executionControl {
			gitClient.PushLocalRepoToEmptyRemote(githubHost, githubOwner, localRepo, remoteName)
		} else {
			log.Println("already hydrated the github gitops repository")
		}
		progressPrinter.IncrementTracker("step-github", 1)

		//* create kubernetes cluster
		executionControl = viper.GetBool("k3d.created") // todo fix this executionControl value `github.detokenized-gitops.pushed`?
		if !executionControl {
			informUser("Creating K8S Cluster", globalFlags.SilentMode)
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

		//* add secrets to cluster
		// todo there is a secret condition in AddK3DSecrets to this not checked
		executionControl = viper.GetBool("kubernetes.atlantis-secrets.secret.created")
		if !executionControl {

			err = k3d.AddK3DSecrets(globalFlags.DryRun)
			if err != nil {
				log.Println("Error AddK3DSecrets")
				return err
			}
		} else {
			log.Println("already added secrets to k3d cluster")
		}

		//* create argocd repository
		informUser("Setup ArgoCD", globalFlags.SilentMode)
		executionControl = viper.GetBool("argocd.initial-repository.created")
		if !executionControl {
			gitopsRepo := fmt.Sprintf("git@github.com:%s/gitops.git", viper.GetString("github.owner"))
			err = argocd.CreateInitialArgoCDRepository(gitopsRepo)
			if err != nil {
				log.Println("Error CreateInitialArgoCDRepository")
				return err
			}
		} else {
			log.Println("already created initial argocd repository")
		}

		//* helm install argocd repository
		informUser("Helm install ArgoCD", globalFlags.SilentMode)
		executionControl = viper.GetBool("argocd.helm-install.complete")
		if !executionControl {
			err = helm.InstallArgocd(globalFlags.DryRun)
			if err != nil {
				log.Println("Error helm installing argocd")
				return err
			}
		} else {
			log.Println("already helm installed argocd")
		}
		progressPrinter.IncrementTracker("step-apps", 1)

		//* argocd pods are ready
		executionControl = viper.GetBool("argocd.ready")
		if !executionControl {
			waitArgoCDToBeReady(globalFlags.DryRun)
			informUser("ArgoCD is ready, continuing", globalFlags.SilentMode)
		} else {
			log.Println("already waited for argocd to be ready")
		}

		//! everything between here

		// todo do we need this again
		kPortForwardArgocd, err = k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
		defer func() {
			err = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardArgocd")
			}
		}()
		informUser(fmt.Sprintf("port-forward to argocd is available at %s", viper.GetString("argocd.local.service")), globalFlags.SilentMode)

		token := "" // todo need to get rid of this and use value from viper
		//* argocd pods are ready
		executionControl = viper.GetBool("argocd.credentials.set")
		if !executionControl {

			informUser("Setting argocd credentials", globalFlags.SilentMode)
			setArgocdCreds(globalFlags.DryRun)
			informUser("Getting an argocd auth token", globalFlags.SilentMode)
			totalAttempts := 3

			if kPortForwardArgocd != nil {
				err = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					log.Println(err)
				}
			}
			for i := 0; i < totalAttempts; i++ {
				kPortForwardArgocd, err = k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
				defer func() {
					err = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
					if err != nil {
						log.Println("Error closing kPortForwardArgocd")
					}
				}()
				token = argocd.GetArgocdAuthToken(globalFlags.DryRun)
				err = argocd.ApplyRegistryLocal(globalFlags.DryRun)
				if err != nil {
					log.Println("Error applying registry")
					return err
				}
			}
		}
		informUser("Syncing the registry application", globalFlags.SilentMode)

		progressPrinter.IncrementTracker("step-apps", 1)

		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, Sync ArgoCD skipped")
		} else {
			// todo: create ArgoCD struct, and host dependencies (like http client)
			customTransport := http.DefaultTransport.(*http.Transport).Clone()
			customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			httpClient := http.Client{Transport: customTransport}

			// retry to sync ArgoCD application until reaches the maximum attempts
			argoCDIsReady, err := argocd.SyncRetry(&httpClient, 20, 5, "registry", token)
			if err != nil {
				log.Printf("something went wrong during ArgoCD sync step, error is: %v", err)
			}

			if !argoCDIsReady {
				log.Println("unable to sync ArgoCD application, continuing...")
			}
		}
		informUser("Setup ArgoCD", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		//! everything between here

		// TODO: K3D => We need to check what changes for vault on raft mode, without terraform to unseal it
		informUser("Waiting vault to be ready", globalFlags.SilentMode)
		waitVaultToBeRunning(globalFlags.DryRun)
		kPortForwardVault, err := k8s.PortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
		defer func() {
			err = kPortForwardVault.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardVault")
			}
		}()

		loopUntilPodIsReady(globalFlags.DryRun)

		informUser("Welcome to local kubefist experience", globalFlags.SilentMode)
		informUser("To use your cluster port-forward - argocd", globalFlags.SilentMode)
		informUser("If not automatically injected, your kubevonfig is at:", globalFlags.SilentMode)
		informUser("k3d kubeconfig get "+viper.GetString("cluster-name"), globalFlags.SilentMode)
		informUser("Expose Argo-CD", globalFlags.SilentMode)
		informUser("kubectl -n argocd port-forward svc/argocd-server 8080:80", globalFlags.SilentMode)
		informUser("Argo User: "+viper.GetString("argocd.admin.username"), globalFlags.SilentMode)
		informUser("Argo Password: "+viper.GetString("argocd.admin.password"), globalFlags.SilentMode)
		time.Sleep(1 * time.Second)
		//TODO: Remove me
		log.Println("Hard break as we are still testing this mode")
		return nil

		if !viper.GetBool("vault.configuredsecret") { //skipVault
			informUser("waiting for vault unseal", globalFlags.SilentMode)
			log.Println("configuring vault")
			// TODO: K3D => I think this may keep working, I think we are just populating vault
			vault.ConfigureVault(globalFlags.DryRun)
			informUser("Vault configured", globalFlags.SilentMode)

			vault.GetOidcClientCredentials(globalFlags.DryRun)
			log.Println("vault oidc clients created")

			log.Println("creating vault configured secret")
			k8s.CreateVaultConfiguredSecret(globalFlags.DryRun, config)
			informUser("Vault secret created", globalFlags.SilentMode)
		}
		informUser("Terraform Vault", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		// TODO: K3D =>  It should work as expected
		directory := fmt.Sprintf("%s/gitops/terraform/users", config.K1FolderPath)
		gitProvider := viper.GetString("git.mode")
		informUser("applying users terraform", globalFlags.SilentMode)
		err = terraform.ApplyUsersTerraform(globalFlags.DryRun, directory, gitProvider)
		if err != nil {
			log.Println(err)
		}
		progressPrinter.IncrementTracker("step-base", 1)
		progressPrinter.IncrementTracker("step-apps", 1)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createGithubK3dCmd)
	currentCommand := createGithubK3dCmd
	flagset.DefineGithubCmdFlags(currentCommand)
	flagset.DefineGlobalFlags(currentCommand)
}
