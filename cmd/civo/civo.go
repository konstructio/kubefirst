package civo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

	// Check quotas
	quotaMessage, quotaFailures, quotaWarnings, err := returnCivoQuotaEvaluation(false)
	if err != nil {
		return err
	}
	switch {
	case quotaFailures > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
		return errors.New("At least one of your Civo quotas is close to its limit. Please check the error message above for additional details.")
	case quotaWarnings > 0:
		fmt.Println(reports.StyleMessage(quotaMessage))
	}

	// printConfirmationScreen()
	// log.Info().Msg("proceeding with cluster create")

	argocdLocalURL := viper.GetString("components.argocd.port-forward-url")
	cloudProvider := viper.GetString("flags.cloud-provider")
	clusterName := viper.GetString("flags.cluster-name")
	clusterType := viper.GetString("flags.cluster-type")
	destinationGitopsRepoURL := viper.GetString("github.repos.gitops.git-url")
	destinationMetaphorRepoURL := viper.GetString("github.repos.metaphor.git-url")
	domainName := viper.GetString("flags.domain-name")
	dryRun := false // todo deprecate this?
	gitopsTemplateBranch := viper.GetString("flags.gitops-template-branch")
	gitopsTemplateURL := viper.GetString("flags.gitops-template-url")
	gitProvider := viper.GetString("flags.git-provider")
	helmClientPath := viper.GetString("k1-paths.helm-client")
	helmClientVersion := viper.GetString("tools.helm.client-version")
	k1Dir := viper.GetString("k1-paths.k1-dir")
	k1GitopsDir := viper.GetString("k1-paths.gitops-dir")
	k1MetaphorDir := viper.GetString("k1-paths.metaphor-dir")
	k1ToolsDir := viper.GetString("k1-paths.tools-dir")
	kubeconfigPath := viper.GetString("k1-paths.kubeconfig")
	kubectlClientPath := viper.GetString("k1-paths.kubectl-client")
	kubectlClientVersion := viper.GetString("tools.kubectl.client-version")
	kubefirstBotSSHPrivateKey := viper.GetString("kbot.private-key")
	localOs := viper.GetString("tools.localhost.os")
	localArchitecture := viper.GetString("tools.localhost.architecture")
	metaphorTemplateBranch := viper.GetString("flags.metaphor-template-branch")
	metaphorTemplateURL := viper.GetString("flags.metaphor-template-url")
	silentMode := false // todo deprecate this?
	terraformClientVersion := viper.GetString("tools.terraform.client-version")

	backupDir := fmt.Sprintf("%s/ssl/%s", k1Dir, domainName)

	//* generate public keys for ssh
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallStarted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

	publicKeys, err := ssh.NewPublicKeys("git", []byte(kubefirstBotSSHPrivateKey), "")
	if err != nil {
		log.Info().Msgf("generate public keys failed: %s\n", err.Error())
	}

	clientset, err := k8s.GetClientSet(dryRun, kubeconfigPath)
	if err != nil {
		return err
	}

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainName, pkg.MetricMgmtClusterInstallStarted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
		}
	}

	//* download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
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
		viper.Set("kubefirst-checks.tools-downloaded", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {
	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {

		pkg.InformUser("generating your new gitops repository", silentMode)
		gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, k1GitopsDir, gitopsTemplateURL)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", k1GitopsDir)
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
		viper.Set("kubefirst-checks.gitops-ready-to-push", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	//* create teams and repositories in github
	executionControl := viper.GetBool("kubefirst-checks.terraform-apply-github")
	if !executionControl {
		pkg.InformUser("Creating github resources with terraform", silentMode)

		tfEntrypoint := k1GitopsDir + "/terraform/github"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating github resources with terraform %s : %s", tfEntrypoint, err))
		}

		pkg.InformUser(fmt.Sprintf("Created git repositories and teams in github.com/%s", githubOwnerFlag), silentMode)
		viper.Set("kubefirst-checks.terraform-apply-github", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created github terraform resources")
	}

	//* push detokenized gitops-template repository content to new remote
	executionControl = viper.GetBool("kubefirst-checks.gitops-repo-pushed")
	if !executionControl {
		gitopsRepo, err := git.PlainOpen(k1GitopsDir)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", k1GitopsDir)
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: gitProvider,
			Auth:       publicKeys,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", destinationGitopsRepoURL)
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/gitops", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		pkg.InformUser(fmt.Sprintf("Created git repositories and teams in github.com/%s", githubOwnerFlag), silentMode)
		viper.Set("kubefirst-checks.gitops-repo-pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already pushed detokenized gitops repository content")
	}

	//* git clone and detokenize the metaphor-frontend-template repository
	if !viper.GetBool("kubefirst-checks.metaphor-repo-pushed") {

		if configs.K1Version != "" {
			gitopsTemplateBranch = configs.K1Version
		}

		pkg.InformUser("generating your new metaphor-frontend repository", silentMode)
		metaphorRepo, err := gitClient.CloneRefSetMain(metaphorTemplateBranch, k1MetaphorDir, metaphorTemplateURL)
		if err != nil {
			log.Info().Msgf("error opening repo at: %s", k1MetaphorDir)
		}

		log.Info().Msg("metaphor repository clone complete")

		err = pkg.CivoGithubAdjustMetaphorTemplateContent(gitProvider, k1Dir, k1MetaphorDir)
		if err != nil {
			return err
		}

		err = pkg.DetokenizeCivoGithubMetaphor(k1MetaphorDir)
		if err != nil {
			return err
		}
		err = gitClient.AddRemote(destinationMetaphorRepoURL, gitProvider, metaphorRepo)
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
			log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", destinationMetaphorRepoURL)
		}

		log.Info().Msgf("successfully pushed gitops to git@github.com/%s/metaphor-frontend", githubOwnerFlag)
		// todo delete the local gitops repo and re-clone it
		// todo that way we can stop worrying about which origin we're going to push to
		pkg.InformUser(fmt.Sprintf("pushed detokenized metaphor-frontend repository to github.com/%s", githubOwnerFlag), silentMode)

		viper.Set("kubefirst-checks.metaphor-repo-pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already completed gitops repo generation - continuing")
	}

	//* create civo cloud resources
	if !viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		pkg.InformUser("Creating civo cloud resources with terraform", silentMode)

		tfEntrypoint := k1GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating civo resources with terraform %s : %s", tfEntrypoint, err))
		}

		pkg.InformUser("Created civo cloud resources", silentMode)
		viper.Set("kubefirst-checks.terraform-apply-civo", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created github terraform resources")
	}

	// kubernetes.BootstrapSecrets
	// todo there is a secret condition in AddK3DSecrets to this not checked
	// todo deconstruct CreateNamespaces / CreateSecret
	// todo move secret structs to constants to be leveraged by either local or civo
	executionControl = viper.GetBool("kubefirst-checks.k8s-secrets-created")
	if !executionControl {
		err := civo.BootstrapCivoMgmtCluster(dryRun, kubeconfigPath)
		if err != nil {
			log.Info().Msg("Error adding kubernetes secrets for bootstrap")
			return err
		}
		viper.Set("kubefirst-checks.k8s-secrets-created", true)
		viper.WriteConfig()

	} else {
		log.Info().Msg("already added secrets to civo cluster")
	}

	//* check for ssl restore
	log.Info().Msg("checking for tls secrets to restore")
	secretsFilesToRestore, err := ioutil.ReadDir(backupDir + "/secrets")
	if err != nil {
		log.Info().Msgf("%s", err)
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
	executionControl = viper.GetBool("kubefirst-checks.argocd-helm-repo-added")
	if !executionControl {
		log.Info().Msgf("helm repo add %s %s and helm repo update", helmRepo.RepoName, helmRepo.RepoURL)
		helm.AddRepoAndUpdateRepo(dryRun, helmClientPath, helmRepo, kubeconfigPath)
		log.Info().Msg("helm repo added")
		viper.Set("kubefirst-checks.argocd-helm-repo-added", true)
		viper.WriteConfig()
	}
	//* helm install argocd
	executionControl = viper.GetBool("kubefirst-checks.argocd-helm-install")
	if !executionControl {
		pkg.InformUser(fmt.Sprintf("helm install %s and wait", helmRepo.RepoName), silentMode)
		// todo adopt golang helm client for helm install
		err := helm.Install(dryRun, helmClientPath, helmRepo, kubeconfigPath)
		if err != nil {
			return err
		}
		viper.Set("kubefirst-checks.argocd-helm-install", true)
		viper.WriteConfig()
	}

	// Wait for ArgoCD StatefulSet Pods to transition to Running
	argoCDStatefulSet, err := k8s.ReturnStatefulSetObject(
		kubeconfigPath,
		"app.kubernetes.io/part-of",
		"argocd",
		"argocd",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding ArgoCD StatefulSet: %s", err)
	}
	_, err = k8s.WaitForStatefulSetReady(kubeconfigPath, argoCDStatefulSet, 90)
	if err != nil {
		log.Info().Msgf("Error waiting for ArgoCD StatefulSet ready state: %s", err)
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
	log.Info().Msgf("port-forward to argocd is available at %s", argocdLocalURL)

	//* argocd pods are ready, get and set credentials
	executionControl = viper.GetBool("kubefirst-checks.argocd-credentials-set")
	if !executionControl {
		log.Info().Msg("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

		argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
		if argocdPassword == "" {
			log.Info().Msg("argocd password not found in secret")
			return err
		}

		viper.Set("components.argocd.password", argocdPassword)
		viper.Set("components.argocd.username", "admin")
		viper.WriteConfig()
		log.Info().Msg("argocd username and password credentials set successfully")

		log.Info().Msg("Getting an argocd auth token")
		// todo return in here and pass argocdAuthToken as a parameter
		token, err := argocd.GetArgoCDToken("admin", argocdPassword)
		if err != nil {
			return err
		}

		log.Info().Msg("argocd admin auth token set")

		viper.Set("components.argocd.auth-token", token)
		viper.Set("kubefirst-checks.argocd-credentials-set", true)
		viper.WriteConfig()
	}

	//* argocd sync registry and start sync waves
	executionControl = viper.GetBool("kubefirst-checks.argocd-create-registry")
	if !executionControl {
		log.Info().Msg("applying the registry application to argocd")
		registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", k1Dir, clusterName)
		_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
		if err != nil {
			log.Warn().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
			return err
		}
		viper.Set("kubefirst-checks.argocd-create-registry", true)
		viper.WriteConfig()
	}

	// Wait for Vault StatefulSet Pods to transition to Running
	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		kubeconfigPath,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding Vault StatefulSet: %s", err)
	}
	_, err = k8s.WaitForStatefulSetReady(kubeconfigPath, vaultStatefulSet, 60)
	if err != nil {
		log.Info().Msgf("Error waiting for Vault StatefulSet ready state: %s", err)
	}

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

	//* configure vault with terraform
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-vault")
	if !executionControl {
		// todo evaluate progressPrinter.IncrementTracker("step-vault", 1)

		//* run vault terraform
		pkg.InformUser("configuring vault with terraform", silentMode)

		tfEnvs := map[string]string{}

		tfEnvs = civo.GetVaultTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEntrypoint := k1GitopsDir + "/terraform/vault"
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}

		pkg.InformUser("vault terraform executed successfully", silentMode)
		viper.Set("kubefirst-checks.terraform-apply-vault", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already executed vault terraform")
	}

	//* create users
	executionControl = viper.GetBool("kubefirst-checks.terraform-apply-users")
	if !executionControl {
		pkg.InformUser("applying users terraform", silentMode)

		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetUsersTerraformEnvs(tfEnvs)
		tfEntrypoint := k1GitopsDir + "/terraform/users"
		err := terraform.InitApplyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			return err
		}
		pkg.InformUser("executed users terraform successfully", silentMode)
		// progressPrinter.IncrementTracker("step-users", 1)
		viper.Set("kubefirst-checks.terraform-apply-users", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("already created users with terraform")
	}

	// Wait for console Deployment Pods to transition to Running
	consoleDeployment, err := k8s.ReturnDeploymentObject(
		kubeconfigPath,
		"app.kubernetes.io/instance",
		"kubefirst-console",
		"kubefirst",
		60,
	)
	if err != nil {
		log.Info().Msgf("Error finding console Deployment: %s", err)
	}
	_, err = k8s.WaitForDeploymentReady(kubeconfigPath, consoleDeployment, 120)
	if err != nil {
		log.Info().Msgf("Error waiting for console Deployment ready state: %s", err)
	}

	//* console port-forward
	consoleStopChannel := make(chan struct{}, 1)
	defer func() {
		close(consoleStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kubeconfigPath,
		"kubefirst-console",
		"kubefirst",
		8080,
		9094,
		consoleStopChannel,
	)

	log.Info().Msg("kubefirst installation complete")
	log.Info().Msg("welcome to your new kubefirst platform powered by Civo cloud")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	reports.LocalHandoffScreen(dryRun, silentMode)

	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(domainNameFlag, pkg.MetricMgmtClusterInstallCompleted, cloudProvider, gitProvider); err != nil {
			log.Info().Msg(err.Error())
			return err
		}
	}

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
	createKubefirstSummary.WriteString(fmt.Sprintf("DNS:    %s\n", viper.GetString("flags.domain-name")))
	createKubefirstSummary.WriteString(fmt.Sprintf("Region: %s\n", viper.GetString("flags.cloud-region")))
	createKubefirstSummary.WriteString("\nGithub Organization Details:\n\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("Organization: %s\n", viper.GetString("github.owner")))
	createKubefirstSummary.WriteString(fmt.Sprintf("User:         %s\n", viper.GetString("github.user")))
	createKubefirstSummary.WriteString("New Github Repository URLs:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repos.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repos.metaphor.url")))

	createKubefirstSummary.WriteString("\nTemplate Repository URLs:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("flags.gitops-template-url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("flags.gitops-template-branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("flags.metaphor-template-url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("flags.metaphor-template-branch")))

	log.Info().Msg(reports.StyleMessage(createKubefirstSummary.String()))
}
