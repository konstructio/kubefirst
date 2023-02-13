package civo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// todo more error handling on function calls

func runCivo(cmd *cobra.Command, args []string) error {

	// config := configs.GetCivoConfig()
	log.Info().Msg("runCivo command is starting ")
	// var userInput string
	// printConfirmationScreen()
	// go counter()
	// fmt.Info().Msg("to proceed, type 'yes' any other answer will exit")
	// fmt.Scanln(&userInput)
	// fmt.Info().Msg("proceeding with cluster create")

	// fmt.Fprintf(w, "%s to open %s in your browser... ", cs.Bold("Press Enter"), oauthHost)
	// https://github.com/cli/cli/blob/trunk/internal/authflow/flow.go#L37
	// to do consider if we can credit github on theirs

	printConfirmationScreen()
	log.Info().Msg("proceeding with cluster create")

	//! viper config variables
	//! should varFlag variations be used in here?
	argocdLocalURL := viper.GetString("argocd.local.service")
	domainName := viper.GetString("domain-name")
	clusterName := viper.GetString("kubefirst.cluster-name")
	clusterType := viper.GetString("kubefirst.cluster-type")
	gitopsTemplateBranch := viper.GetString("template-repo.gitops.branch")
	gitopsTemplateURL := viper.GetString("template-repo.gitops.url")
	metaphorFrontendTemplateBranch := viper.GetString("template-repo.metaphor-frontend.branch")
	metaphorFrontendTemplateURL := viper.GetString("template-repo.metaphor-frontend.url")
	k1MetaphorDir := viper.GetString("kubefirst.k1-metaphor-dir")
	cloudProvider := viper.GetString("cloud-provider")
	destinationGitopsRepoURL := viper.GetString("github.repo.gitops.giturl")
	destinationMetaphorFrontendRepoURL := viper.GetString("github.repo.metaphor-frontend.giturl")
	gitProvider := viper.GetString("git-provider")
	kubefirstBotSSHPrivateKey := viper.GetString("kubefirst.bot.private-key")
	k1GitopsDir := viper.GetString("kubefirst.k1-gitops-dir")
	kubeconfigPath := viper.GetString("kubefirst.kubeconfig-path")
	helmClientPath := viper.GetString("kubefirst.helm-client-path")
	helmClientVersion := viper.GetString("kubefirst.helm-client-version")
	k1Dir := viper.GetString("kubefirst.k1-dir")
	kubectlClientPath := viper.GetString("kubefirst.kubectl-client-path")
	kubectlClientVersion := viper.GetString("kubefirst.kubectl-client-version")
	localOs := viper.GetString("localhost.os")
	localArchitecture := viper.GetString("localhost.architecture")
	terraformClientVersion := viper.GetString("kubefirst.terraform-client-version")
	k1ToolsDir := viper.GetString("kubefirst.k1-tools-dir")
	silentMode := false
	dryRun := false // todo deprecate this?
	backupDir := fmt.Sprintf("%s/ssl/%s", k1Dir, domainName)

	//* create ssl backup directories
	if _, err := os.Stat(backupDir + "/certificates"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		paths := []string{backupDir + "/certificates", backupDir + "/clusterissuers", backupDir + "/secrets"}

		for _, path := range paths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				log.Info().Msgf("checking path: %s", path)
				err := os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Info().Msg("directory already exists, continuing")
				}
			}
		}
	}

	publicKeys, err := ssh.NewPublicKeys("git", []byte(kubefirstBotSSHPrivateKey), "")
	if err != nil {
		log.Info().Msgf("generate publickeys failed: %s\n", err.Error())
	}

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainName, pkg.MetricMgmtClusterInstallStarted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
		}
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst.dependency-download.complete") {
		log.Info().Msg("installing kubefirst dependencies")

		err := downloadManager.CivoDownloadTools(helmClientPath,
			helmClientVersion,
			kubectlClientPath,
			kubectlClientVersion,
			localOs,
			localArchitecture,
			terraformClientVersion,
			k1ToolsDir)
		if err != nil {
			return err
		}

		log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst.dependency-download.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {
	if !viper.GetBool("template-repo.gitops.ready-to-push") {

		pkg.InformUser("generating your new gitops repository", silentMode)
		gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, k1GitopsDir, gitopsTemplateURL)
		if err != nil {
			log.Print("error opening repo at:", k1GitopsDir)
		}
		log.Info().Msg("gitops repository clone complete")

		err = pkg.CivoGithubAdjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1Dir, k1GitopsDir)
		if err != nil {
			return err
		}

		pkg.DetokenizeCivoGithubGitops(k1GitopsDir)
		if err != nil {
			return err
		}
		err = gitClient.AddRemote(destinationGitopsRepoURL, gitProvider, gitopsRepo)
		if err != nil {
			return err
		}

		err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
		if err != nil {
			return err
		}

		// todo emit init telemetry end
		viper.Set("template-repo.gitops.ready-to-push", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	//* create teams and repositories in github
	executionControl := viper.GetBool("terraform.github.apply.complete")
	if !executionControl {
		pkg.InformUser("Creating github resources with terraform", silentMode)

		tfEntrypoint := k1GitopsDir + "/terraform/github"
		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetGithubTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating github resources with terraform %s : %s", tfEntrypoint, err))
		}

		pkg.InformUser(fmt.Sprintf("Created git repositories and teams in github.com/%s", githubOwnerFlag), silentMode)
		viper.Set("terraform.github.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created github terraform resources")
	}

	//* push detokenized gitops-template repository content to new remote
	executionControl = viper.GetBool("github.gitops.repo.pushed")
	if !executionControl {
		gitopsRepo, err := git.PlainOpen(k1GitopsDir)
		if err != nil {
			log.Print("error opening repo at:", k1GitopsDir)
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: gitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", destinationGitopsRepoURL)
		}

		log.Printf("successfully pushed gitops to git@github.com/%s/gitops", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		pkg.InformUser(fmt.Sprintf("Created git repositories and teams in github.com/%s", githubOwnerFlag), silentMode)
		viper.Set("github.gitops.repo.pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
	}

	//* git clone and detokenize the metaphor-frontend-template repository
	if !viper.GetBool("template-repo.metaphor-frontend.pushed") {

		if configs.K1Version != "" {
			gitopsTemplateBranch = configs.K1Version
		}

		pkg.InformUser("generating your new metaphor-frontend repository", silentMode)
		metaphorRepo, err := gitClient.CloneRefSetMain(metaphorFrontendTemplateBranch, k1MetaphorDir, metaphorFrontendTemplateURL)
		if err != nil {
			log.Print("error opening repo at:", k1MetaphorDir)
		}

		fmt.Println("metaphor repository clone complete")

		err = pkg.CivoGithubAdjustMetaphorTemplateContent(gitProvider, k1Dir, k1MetaphorDir)
		if err != nil {
			return err
		}

		err = pkg.DetokenizeCivoGithubMetaphor(k1MetaphorDir)
		if err != nil {
			return err
		}
		err = gitClient.AddRemote(destinationMetaphorFrontendRepoURL, gitProvider, metaphorRepo)
		if err != nil {
			return err
		}

		err = gitClient.Commit(metaphorRepo, "committing detokenized metaphor-frontend-template repo content")
		if err != nil {
			return err
		}

		err = metaphorRepo.Push(&git.PushOptions{
			RemoteName: gitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", destinationMetaphorFrontendRepoURL)
		}

		log.Printf("successfully pushed gitops to git@github.com/%s/metaphor-frontend", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		pkg.InformUser(fmt.Sprintf("pushed detokenized metaphor-frontend repository to github.com/%s", githubOwnerFlag), silentMode)

		viper.Set("template-repo.metaphor-frontend.pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	//* create civo cloud resources
	if !viper.GetBool("terraform.civo.apply.complete") {
		pkg.InformUser("Creating civo cloud resources with terraform", silentMode)

		tfEntrypoint := k1GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetCivoTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating civo resources with terraform %s : %s", tfEntrypoint, err))
		}

		pkg.InformUser("Created civo cloud resources", silentMode)
		viper.Set("terraform.civo.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created github terraform resources")
	}

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	executionControl = viper.GetBool("kubernetes.secrets.created")
	if !executionControl {
		err := civo.BootstrapCivoMgmtCluster(dryRun, kubeconfigPath)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
	} else {
		log.Info().Msg("already added secrets to civo cluster")
	}

	//* check for ssl restore
	log.Info().Msg("checking for tls secrets to restore")
	secretsFilesToRestore, err := ioutil.ReadDir(backupDir + "/secrets")
	if err != nil {
		return err
	}
	if len(secretsFilesToRestore) != 0 {
		// todo would like these but requires CRD's and is not currently supported
		// add crds ( use execShellReturnErrors? )
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
		// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
		// add certificates, and clusterissuers
		log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
		ssl.Restore(backupDir, domainName, kubeconfigPath)
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
	executionControl = viper.GetBool("argocd.helm.repo.updated")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL), silentMode)
		helm.AddRepoAndUpdateRepo(dryRun, helmClientPath, helmRepo, kubeconfigPath)
	}
	//* helm install argocd
	executionControl = viper.GetBool("argocd.helm.install.complete")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), silentMode)
		// todo adopt golang helm client for helm install
		err := helm.Install(dryRun, helmClientPath, helmRepo, kubeconfigPath)
		if err != nil {
			return err
		}
	}

	//* argocd pods are running
	// todo improve this check, also return an error so we can have an exit on failure
	executionControl = viper.GetBool("argocd.ready")
	if !executionControl {
		argocd.WaitArgoCDToBeReady(dryRun, kubeconfigPath, kubectlClientPath)
		pkg.InformUser("ArgoCD is running, continuing", silentMode)
	} else {
		log.Info().Msg("already waited for argocd to be ready")
	}

	//* ArgoCD port-forward
	argoCDStopChannel := make(chan struct{}, 1)
	defer func() {
		close(argoCDStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kubeconfigPath,
		"argocd-server", // todo fix this, it should `argocd
		"argocd",
		8080,
		8080,
		argoCDStopChannel,
	)
	pkg.InformUser(fmt.Sprintf("port-forward to argocd is available at %s", argocdLocalURL), silentMode)

	//* argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("argocd.credentials.set")
	if !executionControl {
		pkg.InformUser("Setting argocd username and password credentials", silentMode)
		k8s.SetArgocdCreds(dryRun, kubeconfigPath)
		pkg.InformUser("argocd username and password credentials set successfully", silentMode)

		pkg.InformUser("Getting an argocd auth token", silentMode)
		// todo return in here and pass argocdAuthToken as a parameter
		_ = argocd.GetArgocdAuthToken(dryRun)
		pkg.InformUser("argocd admin auth token set", silentMode)

		viper.Set("argocd.credentials.set", true)
		viper.WriteConfig()
	}

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("argocd.registry.applied")
	if !executionControl {
		pkg.InformUser("applying the registry application to argocd", silentMode)
		registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", k1Dir, clusterName)
		_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
		if err != nil {
			log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
			return err
		}
	}

	//* vault in running state
	// this condition doesnt work for civo,
	// executionControl = viper.GetBool("vault.status.running")
	// if !executionControl {
	// 	pkg.InformUser("Waiting for vault to be ready", silentMode)
	// 	vault.WaitVaultToBeRunning(dryRun, kubeconfigPath, kubectlClientPath)
	// }
	// todo fix this hack, but vault is unsealed by default in current state
	log.Info().Msg("waiting for vault pods to be running")
	time.Sleep(time.Second * 25)

	//* vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kubeconfigPath,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	//! todo need to pass in url values for connectivity
	// k8s.LoopUntilPodIsReady(dryRun, kubeconfigPath, kubectlClientPath)
	// todo fix this hack, but vault is unsealed by default in current state
	log.Info().Msg("port forward to vault complete")
	time.Sleep(time.Second * 15)

	//* configure vault with terraform
	executionControl = viper.GetBool("terraform.vault.apply.complete")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

		//* run vault terraform
		pkg.InformUser("configuring vault with terraform", silentMode)

		tfEnvs := map[string]string{}

		tfEnvs = terraform.GetVaultTerraformEnvs(tfEnvs)
		tfEnvs = terraform.GetCivoTerraformEnvs(tfEnvs)
		tfEntrypoint := k1GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}

		pkg.InformUser("vault terraform executed successfully", silentMode)

		//* create vault configured secret
		// todo remove this code
		log.Info().Msg("creating vault configured secret")
		k8s.CreateVaultConfiguredSecret(dryRun, kubeconfigPath, kubectlClientPath)
		pkg.InformUser("Vault secret created", silentMode)
		viper.Set("terraform.vault.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already executed vault terraform")
	}

	//* create users
	executionControl = viper.GetBool("terraform.users.apply.complete")
	if !executionControl {
		pkg.InformUser("applying users terraform", silentMode)

		tfEnvs := map[string]string{}
		tfEnvs = terraform.GetCivoTerraformEnvs(tfEnvs)
		tfEnvs = terraform.GetUsersTerraformEnvs(tfEnvs)
		tfEntrypoint := k1GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}
		pkg.InformUser("executed users terraform successfully", silentMode)
		// progressPrinter.IncrementTracker("step-users", 1)
		viper.Set("terraform.users.apply.complete", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created users with terraform")
	}

	log.Info().Msg("Kubefirst installation finished successfully")
	pkg.InformUser("Kubefirst installation finished successfully", silentMode)
	pkg.InformUser("Welcome to civo kubefirst experience", silentMode)
	pkg.InformUser("To use your cluster port-forward - argocd", silentMode)
	pkg.InformUser("If not automatically injected, your kubeconfig is at:", silentMode)
	pkg.InformUser("civo kubeconfig get "+clusterName, silentMode)
	pkg.InformUser("Expose Argo-CD", silentMode)
	pkg.InformUser("kubectl -n argocd port-forward svc/argocd-server 8080:80", silentMode)
	pkg.InformUser("Argo User: "+viper.GetString("argocd.admin.username"), silentMode)
	pkg.InformUser("Argo Password: "+viper.GetString("argocd.admin.password"), silentMode)

	log.Info().Msg("Starting the presentation of console and api for the handoff screen")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	reports.LocalHandoffScreen(dryRun, silentMode)

	log.Info().Msgf("Kubefirst Console available at: %s", pkg.KubefirstConsoleLocalURLCloud)

	return nil
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
	createKubefirstSummary.WriteString(fmt.Sprintf("DNS:    %s\n", viper.GetString("domain-name")))
	createKubefirstSummary.WriteString(fmt.Sprintf("Region: %s\n", viper.GetString("cloud-region")))
	createKubefirstSummary.WriteString("\nGithub Organization Details:\n\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("Organization: %s\n", viper.GetString("github.owner")))
	createKubefirstSummary.WriteString(fmt.Sprintf("User:         %s\n", viper.GetString("github.user")))
	createKubefirstSummary.WriteString("New Github Repository URLs:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor-frontend.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor-go.url")))

	createKubefirstSummary.WriteString("\nTemplate Repository URLs:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.gitops.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor-frontend.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor-frontend.branch")))

	log.Info().Msg(reports.StyleMessage(createKubefirstSummary.String()))
}
