// /*
// Copyright © 2022 NAME HERE <EMAIL ADDRESS>
// */
package cluster

import (
	"fmt"
	"github.com/kubefirst/kubefirst/cmd/cli/tools"
	"github.com/kubefirst/kubefirst/pkg"
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
		tools.RunInfo(cmd, args)
		var kPortForwardArgocd *exec.Cmd
		progressPrinter.AddTracker("step-0", "Process Parameters", 1)
		progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
		progressPrinter.AddTracker("step-base", "Setup base cluster", 2)
		progressPrinter.AddTracker("step-apps", "Install apps to cluster", 5)
		progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), globalFlags.SilentMode)

		progressPrinter.IncrementTracker("step-0", 1)

		if !globalFlags.UseTelemetry {
			pkg.InformUser("Telemetry Disabled", globalFlags.SilentMode)
		}

		executionControl := viper.GetBool("terraform.github.apply.complete")
		//* create github teams in the org and gitops repo
		if !executionControl {
			pkg.InformUser("Creating github resources with terraform", globalFlags.SilentMode)

			tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
			terraform.InitApplyAutoApprove(globalFlags.DryRun, tfEntrypoint)

			pkg.InformUser(fmt.Sprintf("Created gitops Repo in github.com/%s", viper.GetString("github.owner")), globalFlags.SilentMode)
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
			pkg.InformUser(fmt.Sprintf("pushing local detokenized gitops content to new remote github.com/%s", viper.GetString("github.owner")), globalFlags.SilentMode)
			gitClient.PushLocalRepoToEmptyRemote(githubHost, githubOwner, localRepo, remoteName)
		} else {
			log.Println("already hydrated the github gitops repository")
		}
		progressPrinter.IncrementTracker("step-github", 1)

		//* create kubernetes cluster
		executionControl = viper.GetBool("k3d.created")
		if !executionControl {
			pkg.InformUser("Creating K8S Cluster", globalFlags.SilentMode)
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
		executionControl = viper.GetBool("kubernetes.vault.secret.created")
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
			pkg.InformUser("create initial argocd repository", globalFlags.SilentMode)
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
			pkg.InformUser(fmt.Sprintf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL), globalFlags.SilentMode)
			helm.AddRepoAndUpdateRepo(globalFlags.DryRun, helmRepo)
		}

		//* helm install argocd
		executionControl = viper.GetBool("argocd.helm.install.complete")
		if !executionControl {
			pkg.InformUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), globalFlags.SilentMode)
			helm.Install(globalFlags.DryRun, helmRepo)
		}
		progressPrinter.IncrementTracker("step-apps", 1)

		//* argocd pods are running
		executionControl = viper.GetBool("argocd.ready")
		if !executionControl {
			waitArgoCDToBeReady(globalFlags.DryRun)
			pkg.InformUser("ArgoCD is running, continuing", globalFlags.SilentMode)
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
		pkg.InformUser(fmt.Sprintf("port-forward to argocd is available at %s", viper.GetString("argocd.local.service")), globalFlags.SilentMode)

		//* argocd pods are ready, get and set credentials
		executionControl = viper.GetBool("argocd.credentials.set")
		if !executionControl {
			pkg.InformUser("Setting argocd username and password credentials", globalFlags.SilentMode)
			setArgocdCreds(globalFlags.DryRun)
			pkg.InformUser("argocd username and password credentials set successfully", globalFlags.SilentMode)

			pkg.InformUser("Getting an argocd auth token", globalFlags.SilentMode)
			_ = argocd.GetArgocdAuthToken(globalFlags.DryRun)
			pkg.InformUser("argocd admin auth token set", globalFlags.SilentMode)

			viper.Set("argocd.credentials.set", true)
			viper.WriteConfig()
		}

		//* argocd sync registry and start sync waves
		executionControl = viper.GetBool("argocd.registry.applied")
		if !executionControl {
			pkg.InformUser("applying the registry application to argocd", globalFlags.SilentMode)
			err = argocd.ApplyRegistryLocal(globalFlags.DryRun)
			if err != nil {
				log.Println("Error applying registry application to argocd")
				return err
			}
		}

		progressPrinter.IncrementTracker("step-apps", 1)

		//* vault in running state
		executionControl = viper.GetBool("vault.status.running")
		if !executionControl {
			pkg.InformUser("Waiting for vault to be ready", globalFlags.SilentMode)
			waitVaultToBeRunning(globalFlags.DryRun)
			if err != nil {
				log.Println("error waiting for vault to become running")
				return err
			}
		}
		kPortForwardVault, err := k8s.PortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
		defer func() {
			err = kPortForwardVault.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardVault")
			}
		}()

		loopUntilPodIsReady(globalFlags.DryRun)
		kPortForwardMinio, err := k8s.PortForward(globalFlags.DryRun, "minio", "svc/minio", "9000:9000")
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
			pkg.InformUser("configuring vault with terraform", globalFlags.SilentMode)
			tfEntrypoint := config.GitOpsRepoPath + "/terraform/vault"
			terraform.InitApplyAutoApprove(globalFlags.DryRun, tfEntrypoint)

			pkg.InformUser("vault terraform executed successfully", globalFlags.SilentMode)

			//* create vault configurerd secret
			// todo remove this code
			log.Println("creating vault configured secret")
			k8s.CreateVaultConfiguredSecret(globalFlags.DryRun, config)
			pkg.InformUser("Vault secret created", globalFlags.SilentMode)
		} else {
			log.Println("already executed vault terraform")
		}

		//* create users
		executionControl = viper.GetBool("terraform.users.apply.complete")
		if !executionControl {
			pkg.InformUser("applying users terraform", globalFlags.SilentMode)

			tfEntrypoint := config.GitOpsRepoPath + "/terraform/users"
			terraform.InitApplyAutoApprove(globalFlags.DryRun, tfEntrypoint)

			pkg.InformUser("executed users terraform successfully", globalFlags.SilentMode)
			// progressPrinter.IncrementTracker("step-users", 1)
		} else {
			log.Println("already created users with terraform")
		}

		// TODO: K3D =>  NEED TO REMOVE local-backend.tf and rename remote-backend.md

		pkg.InformUser("Welcome to local kubefirst experience", globalFlags.SilentMode)
		pkg.InformUser("To use your cluster port-forward - argocd", globalFlags.SilentMode)
		pkg.InformUser("If not automatically injected, your kubeconfig is at:", globalFlags.SilentMode)
		pkg.InformUser("k3d kubeconfig get "+viper.GetString("cluster-name"), globalFlags.SilentMode)
		pkg.InformUser("Expose Argo-CD", globalFlags.SilentMode)
		pkg.InformUser("kubectl -n argocd port-forward svc/argocd-server 8080:80", globalFlags.SilentMode)
		pkg.InformUser("Argo User: "+viper.GetString("argocd.admin.username"), globalFlags.SilentMode)
		pkg.InformUser("Argo Password: "+viper.GetString("argocd.admin.password"), globalFlags.SilentMode)
		time.Sleep(1 * time.Second)
		progressPrinter.IncrementTracker("step-apps", 1)
		progressPrinter.IncrementTracker("step-base", 1)
		progressPrinter.IncrementTracker("step-apps", 1)
		return nil
	},
}

//func initialization() {
//	cmd.clusterCmd.AddCommand(createGithubK3dCmd)
//	currentCommand := createGithubK3dCmd
//	flagset.DefineGithubCmdFlags(currentCommand)
//	flagset.DefineGlobalFlags(currentCommand)
//}
