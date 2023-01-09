package civo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	zlog "github.com/rs/zerolog/log"

	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/metaphor"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// todo config. should not be referenced outside of validate, if its a value that must be
// todo generated beyond that we should put it in viper alone

type KubefirstToolConfig struct {
	namespace string
	name      string
}

func runCivo(cmd *cobra.Command, args []string) error {

	config := configs.GetCivoConfig()
	log.Println("runCivo command is starting ")
	// var userInput string
	// printConfirmationScreen()
	// go counter()
	// fmt.Println("to proceed, type 'yes' any other answer will exit")
	// fmt.Scanln(&userInput)
	// fmt.Println("proceeding with cluster create")

	// fmt.Fprintf(w, "%s to open %s in your browser... ", cs.Bold("Press Enter"), oauthHost)
	// https://github.com/cli/cli/blob/trunk/internal/authflow/flow.go#L37
	// to do consider if we can credit github on theirs

	printConfirmationScreen()
	fmt.Println("proceeding with cluster create")

	//! viper config variables
	civoDnsName := viper.GetString("civo.dns")
	gitopsTemplateBranch := viper.GetString("template-repo.gitops.branch")
	gitopsTemplateURL := viper.GetString("template-repo.gitops.url")
	cloudProvider := viper.GetString("cloud-provider")
	gitProvider := viper.GetString("git-provider")
	kubeconfigPath := viper.GetString("kubefirst.kubeconfig-path")
	helmClientPath := viper.GetString("kubefirst.helm-client-path")
	helmClientVersion := viper.GetString("kubefirst.helm-client-version")
	k1DirectoryPath := viper.GetString("kubefirst.k1-directory-path")
	kubectlClientPath := viper.GetString("kubefirst.kubectl-client-path")
	kubectlClientVersion := viper.GetString("kubefirst.kubectl-client-version")
	localOs := viper.GetString("localhost.os")
	localArchitecture := viper.GetString("localhost.architecture")
	terraformClientVersion := viper.GetString("kubefirst.terraform-client-version")
	k1ToolsDirPath := viper.GetString("kubefirst.k1-tools-path")
	silentMode := false // todo fix
	dryRun := false     // todo fix

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(civoDnsName, pkg.MetricMgmtClusterInstallStarted); err != nil {
			log.Println(err)
		}
	}

	//* download dependencies `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst.dependency-download.complete") {
		log.Println("installing kubefirst dependencies")

		err := downloadManager.CivoDownloadTools(helmClientPath,
			helmClientVersion,
			kubectlClientPath,
			kubectlClientVersion,
			localOs,
			localArchitecture,
			terraformClientVersion,
			k1ToolsDirPath)
		if err != nil {
			return err
		}

		log.Println("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst.dependency-download.complete", true)
		viper.WriteConfig()
	} else {
		log.Println("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}

	//* git clone and detokenize the gitops repository
	if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {

		//* step 1 clone the gitops-template repository
		pkg.InformUser("generating your new gitops repository", silentMode)
		gitClient.CloneBranchSetMain(gitopsTemplateURL, config.GitOpsRepoPath, gitopsTemplateBranch)
		log.Println("gitops repository clone complete")

		//* step 2 get the correct driver content
		// adjust content in gitops repository
		// clear out the root of `gitops-template` once we move
		// all the content we only remove the different root folders
		os.RemoveAll(config.GitOpsRepoPath + "/components")
		os.RemoveAll(config.GitOpsRepoPath + "/localhost")
		os.RemoveAll(config.GitOpsRepoPath + "/registry")
		os.RemoveAll(config.GitOpsRepoPath + "/validation")
		os.RemoveAll(config.GitOpsRepoPath + "/terraform")
		os.RemoveAll(config.GitOpsRepoPath + "/.gitignore")
		os.RemoveAll(config.GitOpsRepoPath + "/LICENSE")
		os.RemoveAll(config.GitOpsRepoPath + "/README.md")
		os.RemoveAll(config.GitOpsRepoPath + "/atlantis.yaml")
		os.RemoveAll(config.GitOpsRepoPath + "/logo.png")

		driverContent := fmt.Sprintf("%s/%s-%s", config.GitOpsRepoPath, cloudProvider, gitProvider)
		opt := cp.Options{
			Skip: func(src string) (bool, error) {
				if strings.HasSuffix(src, ".git") {
					return true, nil
				} else if strings.Index(src, "/.terraform") > 0 {
					return true, nil
				}
				//Add more stuff to be ignored here
				return false, nil

			},
		}
		err := cp.Copy(driverContent, config.GitOpsRepoPath, opt)
		if err != nil {
			log.Println("Error populating gitops with local setup:", err)
			return err
		}
		os.RemoveAll(driverContent)

		//* step 3 detokenize the new gitops repo driver content
		pkg.DetokenizeCivoGithub(config.GitOpsRepoPath)

		//* step 4 add a new remote of the github user who's token we have
		repo, err := git.PlainOpen(config.GitOpsRepoPath)
		if err != nil {
			log.Print("error opening repo at:", config.GitOpsRepoPath)
		}
		destinationGitopsRepoURL := viper.GetString("github.repo.gitops.giturl")
		log.Printf("git remote add github %s", destinationGitopsRepoURL)
		_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "github",
			URLs: []string{destinationGitopsRepoURL},
		})
		if err != nil {
			log.Panicf("Error creating remote %s at: %s - %s", viper.GetString("git-provider"), destinationGitopsRepoURL, err)
		}

		//* step 5 commit newly detokenized content
		w, _ := repo.Worktree()

		log.Printf("committing detokenized %s content", "gitops")
		status, err := w.Status()
		if err != nil {
			log.Println("error getting worktree status", err)
		}

		for file, _ := range status {
			_, err = w.Add(file)
			if err != nil {
				log.Println("error getting worktree status", err)
			}
		}
		w.Commit(fmt.Sprintf("[ci skip] committing initial detokenized %s content", destinationGitopsRepoURL), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "kubefirst-bot",
				Email: "kubefirst-bot@kubefirst.com",
				When:  time.Now(),
			},
		})

		// todo emit init telemetry end

		viper.Set("template-repo.gitops.cloned", true)
		viper.Set("template-repo.gitops.detokenized", true)
		viper.WriteConfig()
	} else {
		log.Println("already completed gitops repo generation - continuing")
	}

	// todo need to verify only creating gitops and metaphor-frontend
	executionControl := viper.GetBool("terraform.github.apply.complete")
	// create github teams in the org and gitops repo
	if !executionControl {
		pkg.InformUser("Creating github resources with terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetGithubTerraformEnvs(tfEnvs)
		//* only log on debug
		log.Println("tf env vars: ", tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating github resources with terraform %s : %s", tfEntrypoint, err))
		}

		pkg.InformUser(fmt.Sprintf("Created git repositories and teams in github.com/%s", viper.GetString("github.owner")), silentMode)
		viper.Set("terraform.github.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Println("already created github terraform resources")
	}

	//!
	executionControl = viper.GetBool("github.gitops.repo.pushed")
	// create github teams in the org and gitops repo
	if !executionControl {
		//* step 7 push the detokenized gitops repo content
		repo, err := git.PlainOpen(config.GitOpsRepoPath)
		if err != nil {
			log.Print("error opening repo at:", config.GitOpsRepoPath)
		}

		publicKeys, err := ssh.NewPublicKeys("git", []byte(viper.GetString("kubefirst.bot.private-key")), "")
		if err != nil {
			zlog.Info().Msgf("generate publickeys failed: %s\n", err.Error())
		}

		err = repo.Push(&git.PushOptions{
			RemoteName: viper.GetString("git-provider"),
			Auth:       publicKeys,
		})
		if err != nil {
			zlog.Panic().Msgf("error pushing detokenized %s repository to remote at %s", "gitops", viper.GetString("git-provider"))
		}

		log.Printf("successfully pushed gitops to github.com/%s/gitops", viper.GetString("github.owner"))
		//todo delete the local gitops repo and re-clone it, that way we can stop worrying about which origin we're going to push to
		pkg.InformUser(fmt.Sprintf("Created git repositories and teams in github.com/%s", viper.GetString("github.owner")), silentMode)
		viper.Set("github.gitops.repo.pushed", true)
		viper.WriteConfig()
	} else {
		log.Println("already pushed detokenized gitops repository content")
	}

	//!

	// create civo cloud resources
	if !viper.GetBool("terraform.civo.apply.complete") {
		pkg.InformUser("Creating civo cloud resources with terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetCivoTerraformEnvs(tfEnvs)
		//* only log on debug
		log.Println("tf env vars: ", tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating civo resources with terraform %s : %s", tfEntrypoint, err))
		}

		pkg.InformUser("Created civo cloud resources", silentMode)
		viper.Set("terraform.civo.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Println("already created github terraform resources")
	}

	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	executionControl = viper.GetBool("kubernetes.secrets.created")
	if !executionControl {
		err := k3d.AddK3DSecrets(dryRun, kubeconfigPath)
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
		gitopsRepo := viper.GetString("github.repo.gitops.giturl")
		argoCDConfig := argocd.GetArgoCDInitialCloudConfig(gitopsRepo, viper.GetString("kubefirst.bot.private-key"))
		err := argocd.CreateInitialArgoCDRepository(argoCDConfig, k1DirectoryPath)
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

	// argocd pods are running
	executionControl = viper.GetBool("argocd.ready")
	if !executionControl {
		argocd.WaitArgoCDToBeReady(dryRun)
		pkg.InformUser("ArgoCD is running, continuing", silentMode)
	} else {
		log.Println("already waited for argocd to be ready")
	}
	//!
	//!HERE
	//!
	//!
	// ArgoCD port-forward
	argoCDStopChannel := make(chan struct{}, 1)
	defer func() {
		close(argoCDStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
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
		k8s.SetArgocdCreds(dryRun, kubeconfigPath)
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

	// vault in running state
	executionControl = viper.GetBool("vault.status.running")
	if !executionControl {
		pkg.InformUser("Waiting for vault to be ready", silentMode)
		vault.WaitVaultToBeRunning(dryRun, config.KubeConfigPath)
	}

	// Vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		pkg.VaultPodName,
		pkg.VaultNamespace,
		pkg.VaultPodPort,
		pkg.VaultPodLocalPort,
		vaultStopChannel,
	)

	k8s.LoopUntilPodIsReady(dryRun, kubeconfigPath, kubectlClientPath)

	minioStopChannel := make(chan struct{}, 1)
	defer func() {
		close(minioStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		pkg.MinioPodName,
		pkg.MinioNamespace,
		pkg.MinioPodPort,
		pkg.MinioPodLocalPort,
		minioStopChannel,
	)

	// todo: can I remove it?
	time.Sleep(20 * time.Second)

	//! need to look hard starting here down
	//! todo

	// configure vault with terraform
	executionControl = viper.GetBool("terraform.vault.apply.complete")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)
		//* set known vault token
		viper.Set("vault.token", "k1_local_vault_token")
		viper.WriteConfig()

		//* run vault terraform
		pkg.InformUser("configuring vault with terraform", silentMode)

		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetVaultTerraformEnvs(tfEnvs)
		tfEntrypoint := config.GitOpsRepoPath + "/terraform/vault"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)

		pkg.InformUser("vault terraform executed successfully", silentMode)

		//* create vault configurerd secret
		// todo remove this code
		log.Println("creating vault configured secret")
		k8s.CreateVaultConfiguredSecret(dryRun, kubeconfigPath, kubectlClientPath)
		pkg.InformUser("Vault secret created", silentMode)
	} else {
		log.Println("already executed vault terraform")
	}

	// create users
	executionControl = viper.GetBool("terraform.users.apply.complete")
	if !executionControl {
		pkg.InformUser("applying users terraform", silentMode)

		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetUsersTerraformEnvs(tfEnvs)
		tfEntrypoint := config.GitOpsRepoPath + "/terraform/users"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)

		pkg.InformUser("executed users terraform successfully", silentMode)
		// progressPrinter.IncrementTracker("step-users", 1)
	} else {
		log.Println("already created users with terraform")
	}

	pkg.InformUser("Welcome to civo kubefirst experience", silentMode)
	pkg.InformUser("To use your cluster port-forward - argocd", silentMode)
	pkg.InformUser("If not automatically injected, your kubeconfig is at:", silentMode)
	pkg.InformUser("k3d kubeconfig get "+viper.GetString("kubefirst.cluster-name"), silentMode)
	pkg.InformUser("Expose Argo-CD", silentMode)
	pkg.InformUser("kubectl -n argocd port-forward svc/argocd-server 8080:80", silentMode)
	pkg.InformUser("Argo User: "+viper.GetString("argocd.admin.username"), silentMode)
	pkg.InformUser("Argo Password: "+viper.GetString("argocd.admin.password"), silentMode)

	// progressPrinter.IncrementTracker("step-apps", 1)
	// progressPrinter.IncrementTracker("step-base", 1)
	// progressPrinter.IncrementTracker("step-apps", 1)

	if !viper.GetBool("chartmuseum.host.resolved") {
		// Chartmuseum port-forward
		chartmuseumStopChannel := make(chan struct{}, 1)
		defer func() {
			close(chartmuseumStopChannel)
		}()
		k8s.OpenPortForwardPodWrapper(
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
	metaphorBranch := viper.GetString("template-repo.metaphor.branch")
	err := metaphor.DeployMetaphorGithubLocal(dryRun, false, githubOwner, metaphorBranch, "")
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

	githubHost := viper.GetString("github.host")

	localRepo := "gitops"
	remoteName := "github"
	gitopsRepo := "gitops"

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
		k8s.OpenPortForwardPodWrapper(
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
			gitopsRepo,
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

	// if useTelemetry {
	// 	if err = wrappers.SendSegmentIoTelemetry("", pkg.MetricMgmtClusterInstallCompleted); err != nil {
	// 		log.Println(err)
	// 	}
	// 	progressPrinter.IncrementTracker("step-telemetry", 1)
	// }

	log.Println("Kubefirst installation finished successfully")
	pkg.InformUser("Kubefirst installation finished successfully", silentMode)

	// waiting GitHub/atlantis step
	wg.Wait()

	// //! terraform entrypoints

	return errors.New("NO ERROR - we made it to the end, next item")
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}

// todo move below functions? pkg? rename?
func counter() {
	i := 0
	for {
		time.Sleep(time.Second * 1)
		i++
	}
}

func printConfirmationScreen() {
	var createKubefirstSummary bytes.Buffer
	createKubefirstSummary.WriteString(strings.Repeat("-", 70))
	createKubefirstSummary.WriteString("\nCreate Kubefirst Cluster?\n")
	createKubefirstSummary.WriteString(strings.Repeat("-", 70))
	createKubefirstSummary.WriteString("\nCivo Details:\n\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("DNS:    %s\n", viper.GetString("civo.dns")))
	createKubefirstSummary.WriteString(fmt.Sprintf("Region: %s\n", viper.GetString("civo.region")))
	createKubefirstSummary.WriteString("\nGithub Organization Details:\n\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("Organization: %s\n", viper.GetString("github.owner")))
	createKubefirstSummary.WriteString(fmt.Sprintf("User:         %s\n", viper.GetString("github.user")))
	createKubefirstSummary.WriteString("New Github Repository URL's:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor-frontend.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor-go.url")))

	createKubefirstSummary.WriteString("\nTemplate Repository URL's:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.gitops.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor-frontend.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor-frontend.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor-go.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor-go.branch")))

	fmt.Println(reports.StyleMessage(createKubefirstSummary.String()))
}
