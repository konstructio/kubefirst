/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
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
		progressPrinter.AddTracker("step-apps", "Install apps to cluster", 5)
		progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), globalFlags.SilentMode)

		progressPrinter.IncrementTracker("step-0", 1)

		if !globalFlags.UseTelemetry {
			informUser("Telemetry Disabled", globalFlags.SilentMode)
		}

		executionControl := viper.GetBool("terraform.github-k3d.apply.complete")
		//* create github teams in the org and gitops repo
		if !executionControl {
			informUser("Creating github resources with terraform", globalFlags.SilentMode)

			tfEntrypoint := config.GitOpsRepoPath + "/terraform/github-k3d"
			terraform.InitApplyAutoApprove(globalFlags.DryRun, tfEntrypoint)

			informUser(fmt.Sprintf("Created gitops Repo in github.com/%s", viper.GetString("github.owner")), globalFlags.SilentMode)
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
			informUser(fmt.Sprintf("pushing local detokenized gitops content to new remote github.com/%s", viper.GetString("github.owner")), globalFlags.SilentMode)
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

		//* create argocd intiial repository config
		executionControl = viper.GetBool("argocd.initial-repository.created")
		if !executionControl {
			informUser("create initial argocd repository", globalFlags.SilentMode)
			gitopsRepo := fmt.Sprintf("git@github.com:%s/gitops.git", viper.GetString("github.owner"))
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
			informUser(fmt.Sprintf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL), globalFlags.SilentMode)
			helm.AddRepoAndUpdateRepo(globalFlags.DryRun, helmRepo)
		}

		//* helm install argocd
		executionControl = viper.GetBool("argocd.helm.install.complete")
		if !executionControl {
			informUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), globalFlags.SilentMode)
			helm.Install(globalFlags.DryRun, helmRepo)
		}
		progressPrinter.IncrementTracker("step-apps", 1)

		//* argocd pods are running
		executionControl = viper.GetBool("argocd.ready")
		if !executionControl {
			waitArgoCDToBeReady(globalFlags.DryRun)
			informUser("ArgoCD is running, continuing", globalFlags.SilentMode)
		} else {
			log.Println("already waited for argocd to be ready")
		}

		//* establish port-forward
		kPortForwardArgocd, err = k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
		defer func() {
			err = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardArgocd")
			}
		}()
		informUser(fmt.Sprintf("port-forward to argocd is available at %s", viper.GetString("argocd.local.service")), globalFlags.SilentMode)

		//* argocd pods are ready, get and set credentials
		executionControl = viper.GetBool("argocd.credentials.set")
		if !executionControl {
			informUser("Setting argocd username and password credentials", globalFlags.SilentMode)
			setArgocdCreds(globalFlags.DryRun)
			informUser("argocd username and password credentials set successfully", globalFlags.SilentMode)

			informUser("Getting an argocd auth token", globalFlags.SilentMode)
			_ = argocd.GetArgocdAuthToken(globalFlags.DryRun)
			informUser("argocd admin auth token set", globalFlags.SilentMode)

			viper.Set("argocd.credentials.set", true)
			viper.WriteConfig()
		}

		//* argocd sync registry and start sync waves
		executionControl = viper.GetBool("argocd.registry.applied")
		if !executionControl {
			informUser("applying the registry application to argocd", globalFlags.SilentMode)
			err = argocd.ApplyRegistryLocal(globalFlags.DryRun)
			if err != nil {
				log.Println("Error applying registry application to argocd")
				return err
			}
		}

		progressPrinter.IncrementTracker("step-apps", 1)

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
