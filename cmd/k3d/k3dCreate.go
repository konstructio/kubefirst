/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/atotto/clipboard"
	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kubefirst/kubefirst-api/pkg/handlers"
	"github.com/kubefirst/kubefirst-api/pkg/reports"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	utils "github.com/kubefirst/kubefirst-api/pkg/utils"
	"github.com/kubefirst/kubefirst-api/pkg/wrappers"
	"github.com/kubefirst/kubefirst/internal/catalog"
	"github.com/kubefirst/kubefirst/internal/gitShim"
	"github.com/kubefirst/kubefirst/internal/prechecks"
	"github.com/kubefirst/kubefirst/internal/segment"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/argocd"
	"github.com/kubefirst/runtime/pkg/gitClient"
	"github.com/kubefirst/runtime/pkg/github"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/kubefirst/runtime/pkg/progressPrinter"
	"github.com/kubefirst/runtime/pkg/services"
	"github.com/kubefirst/runtime/pkg/terraform"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// required portforwarding ports
	portForwardingPorts = []int{8080, 8200, 9000, 9094}

	// Supported git providers
	supportedGitProviders = []string{"github", "gitlab"}

	// Supported git providers
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

type createOptions struct {
	ci bool

	cloudRegion string
	clusterName string
	clusterType string
	clusterId   string

	containerRegistryHost string

	githubUser string
	githubOrg  string

	gitlabGroup   string
	gitlabGroupId int

	gitProvider   string
	gitProtocol   string
	gitToken      string
	gitHost       string
	gitUser       string
	gitOwner      string
	gitDescriptor string

	gitopsTemplateURL    string
	gitopsTemplateBranch string
	gitopsRepoURL        string

	useTelemetry bool
	segClient    telemetry.TelemetryEvent

	httpClient *http.Client

	installCatalogApps string
	catalogApps        []types.GitopsCatalogApp
}

func defaultCreateOpts() *createOptions {
	return &createOptions{
		clusterName: "kubefirst",
		clusterType: "mgmt",

		gitProvider: "github",
		gitProtocol: "ssh",

		gitopsTemplateURL:    "https://github.com/kubefirst/gitops-template.git",
		gitopsTemplateBranch: "main",

		useTelemetry: true,

		httpClient: http.DefaultClient,

		segClient: telemetry.TelemetryEvent{},
	}
}

func NewK3dCreateCommand() *cobra.Command {
	opts := defaultCreateOpts()

	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running in k3d on your localhost",
		TraverseChildren: true,
		SilenceErrors:    true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// example implementation of a pre check suite where we could validate & verify various required settings (Github access, env vars, commands, ...)
			fmt.Println("[PRECHECKS] Running prechecks")

			// invalid git provider
			if !utilities.StringInSlice(opts.gitProvider, supportedGitProviders) {
				return fmt.Errorf("\"%s\" is not a supported git provider", opts.gitProvider)
			}

			// invalid git protocol
			if !utilities.StringInSlice(opts.gitProtocol, supportedGitProtocolOverride) {
				return fmt.Errorf("%s is not a support git protocol", opts.gitProtocol)
			}

			// github specific prechecks
			if strings.ToLower(opts.gitProvider) == "github" {
				// enforce for GITHUB_TOKEN
				if !prechecks.EnvVarExists("GITHUB_TOKEN") {
					return fmt.Errorf("GITHUB_TOKEN not set, but required when using GitHub (see https://docs.kubefirst.io/common/gitAuth?git_provider=github)")
				}

				// github.com is available
				if err := prechecks.URLIsAvailable("github.com:443"); err != nil {
					return fmt.Errorf("github.com is not available: %s, try again", err.Error())
				}

				// check for known hosts of github.com if git provider is github
				if err := prechecks.CheckKnownHosts("github.com"); err != nil && strings.ToLower(opts.gitProvider) == "github" {
					return err
				}
			}

			// gitlab specific prechecks
			if strings.ToLower(opts.gitProvider) == "gitlab" {
				// enforce GITLAB_TOKEN if provider is gitlab
				if !prechecks.EnvVarExists("GITLAB_TOKEN") {
					return fmt.Errorf("GITLAB_TOKEN not set, but required when using GitLab (see https://docs.kubefirst.io/common/gitAuth)")
				}

				// gitlab.com is available
				if err := prechecks.URLIsAvailable("gitlab.com:443"); err != nil {
					return fmt.Errorf("gitlab.com is not available: %s, try again", err.Error())
				}

				// when gitlab, check for a specified group
				if opts.gitlabGroup == "" {
					return fmt.Errorf("a gitlab-group is required when using Gitlab")
				}

				// check known hosts for gitlab
				if err := prechecks.CheckKnownHosts("gitlab.com"); err != nil && strings.ToLower(opts.gitProvider) == "gitlab" {
					return err
				}
			}

			// enforce NGROK_AUTH_TOKEN
			if !prechecks.EnvVarExists("NGROK_AUTHTOKEN") {
				return fmt.Errorf("NGROK_AUTHTOKEN not set, but required (see https://docs.kubefirst.io/k3d/quick-start/install#local-atlantis-executions-optional)")
			}

			// docker is installed
			if err := prechecks.CommandExists("docker"); err != nil {
				return fmt.Errorf("docker is not installed, but is required when using k3d")
			}

			// check docker is running
			if err := prechecks.CheckDockerIsRunning(); err != nil {
				return fmt.Errorf("docker is not running, but is required when using k3d")
			}

			// portforwarding ports are available
			if err := k8s.CheckForExistingPortForwards(portForwardingPorts...); err != nil {
				return fmt.Errorf("%s - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing", err.Error())
			}

			// verify user has specified valid catalog apps to be installed
			isValid, apps, err := catalog.ValidateCatalogApps(opts.installCatalogApps)
			if !isValid || err != nil {
				return err
			}

			opts.catalogApps = apps

			// check available disk space
			if err := prechecks.CheckAvailableDiskSize(); err != nil {
				return fmt.Errorf("error checking available disk size: %s", err.Error())
			}

			fmt.Println("[PRECHECKS] all prechecks passed - continuing with k3d cluster bootstrapping")

			return nil
		},
		RunE: opts.runK3d,
	}

	// flags
	createCmd.Flags().BoolVar(&opts.ci, "ci", opts.ci, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&opts.clusterName, "cluster-name", opts.clusterName, "the name of the cluster to create")
	createCmd.Flags().StringVar(&opts.clusterType, "cluster-type", opts.clusterType, "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&opts.gitProvider, "git-provider", opts.gitProvider, fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&opts.gitProtocol, "git-protocol", opts.gitProtocol, fmt.Sprintf("the git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&opts.githubUser, "github-user", opts.githubUser, "the GitHub user for the new gitops and metaphor repositories - this cannot be used with --github-org")
	createCmd.Flags().StringVar(&opts.githubOrg, "github-org", opts.githubOrg, "the GitHub organization for the new gitops and metaphor repositories - this cannot be used with --github-user")
	createCmd.Flags().StringVar(&opts.gitlabGroup, "gitlab-group", opts.gitlabGroup, "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&opts.gitopsTemplateBranch, "gitops-template-branch", opts.gitopsTemplateBranch, "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&opts.gitopsTemplateURL, "gitops-template-url", opts.gitopsTemplateURL, "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&opts.installCatalogApps, "install-catalog-apps", opts.installCatalogApps, "comma seperated values of catalog apps to install after provision")
	createCmd.Flags().BoolVar(&opts.useTelemetry, "use-telemetry", opts.useTelemetry, "whether to emit telemetry")

	// flag constraints
	// only one of --github-user or --github-org can be supplied"
	createCmd.MarkFlagsMutuallyExclusive("github-user", "github-org")

	return createCmd
}

func (o *createOptions) runK3d(cmd *cobra.Command, args []string) error {
	// gen clusterID if none exists
	o.clusterId = viper.GetString("kubefirst.cluster-id")
	if o.clusterId == "" {
		o.clusterId = pkg.GenerateClusterID()

		viper.Set("kubefirst.cluster-id", o.clusterId)

		if err := viper.WriteConfig(); err != nil {
			log.Fatal().Msgf("cannot save state: %s", err.Error())
		}
	}

	// configure telemetry, AFAIK will the sending of the metrics event only disabled by setting the USE_TELEMETRY env var
	// so we set that one manually to allow disabling it via the provided flag (see https://github.com/kubefirst/metrics-client/blob/main/pkg/telemetry/telemetry.go)
	if o.useTelemetry {
		if err := os.Setenv("USE_TELEMETRY", fmt.Sprintf("%v", o.useTelemetry)); err != nil {
			log.Warn().Msgf("error setting USE_TELEMETRY env var: %s", err.Error())
		}

		o.segClient = segment.InitClient(o.clusterId, o.clusterType, o.gitProvider)
	}

	// create k1 dir
	utilities.CreateDirIfNotExists(path.Join("~", ".k1", o.clusterName))

	// display logfile
	helpers.DisplayLogHints()

	// Store flags for application state maintenance
	viper.Set("flags.cluster-name", o.clusterName)
	viper.Set("flags.domain-name", k3d.DomainName)
	viper.Set("flags.git-provider", o.gitProvider)
	viper.Set("flags.git-protocol", o.gitProtocol)
	viper.Set("kubefirst.cloud-provider", "k3d")

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	// github specific auth settings
	if strings.ToLower(o.gitProvider) == "github" {
		if err := o.githubAuth(); err != nil {
			return fmt.Errorf("error during github auth: %s", err.Error())
		}
	}

	// gitlab specific auth settings
	if strings.ToLower(o.gitProvider) == "gitlab" {
		if err := o.gitlabAuth(); err != nil {
			return fmt.Errorf("error during gitlab auth: %s", err.Error())
		}
	}

	// Instantiate K3d config
	config := k3d.GetConfig(o.clusterName, o.gitProvider, o.gitOwner, o.gitProtocol)

	// add token to the specified provider
	// TODO: should be done in k3d.GetConfig()
	if strings.ToLower(o.gitProvider) == "github" {
		config.GithubToken = o.gitToken
	}

	if strings.ToLower(o.gitProvider) == "gitlab" {
		config.GitlabToken = o.gitToken
	}

	// TODO: should be done in k3d.GetConfig()
	o.gitopsRepoURL = config.DestinationGitopsRepoGitURL
	if config.GitProtocol == "https" {
		o.gitopsRepoURL = config.DestinationGitopsRepoURL
	}

	progressPrinter.AddTracker("preflight-checks", "Running preflight checks", 5)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)
	progressPrinter.IncrementTracker("preflight-checks", 1)

	// this branch flag value is overridden with a tag when running from a kubefirst binary for version compatibility
	// TODO: review
	// if strings.Contains(o.gitopsTemplateURL, "https://github.com/kubefirst/gitops-template") &&
	// 	o.gitopsTemplateBranch == "" {
	// 	o.gitopsTemplateBranch = configs.K1Version
	// }

	log.Info().Msgf("kubefirst version configs.K1Version: %s ", configs.K1Version)
	log.Info().Msgf("cloning gitops-template repo url: %s ", o.gitopsTemplateURL)
	log.Info().Msgf("cloning gitops-template repo branch: %s ", o.gitopsTemplateBranch)

	log.Info().Msg("checking authentication to required providers")
	progressPrinter.IncrementTracker("preflight-checks", 1)

	// Check git credentials
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider)) {
		if err := o.CheckGitCredentials(config); err != nil {
			return fmt.Errorf("error checking git credentials: %s", err.Error())
		}
	}

	log.Info().Msg(fmt.Sprintf("completed %s checks - continuing", config.GitProvider))

	progressPrinter.IncrementTracker("preflight-checks", 1)

	// setup kbot
	if !viper.GetBool("kubefirst-checks.kbot-setup") {
		if err := o.setupKbot(); err != nil {
			return fmt.Errorf("error setting up kbot: %s", err.Error())
		}
	}

	log.Info().Msg("setup kbot user - continuing")

	progressPrinter.IncrementTracker("preflight-checks", 1)

	log.Info().Msg("validation and kubefirst cli environment check is complete")

	if e := telemetry.SendEvent(o.segClient, telemetry.InitCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	// download dependencies to `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst-checks.tools-downloaded") {
		if err := o.downnloadTools(config); err != nil {
			return fmt.Errorf("error downloading tools: %s", err.Error())
		}
	}

	log.Info().Msg("completed download of dependencies to `$HOME/.k1/tools` - continuing")

	progressPrinter.IncrementTracker("preflight-checks", 1)

	//* git clone and detokenize the gitops repository
	// todo improve this logic for removing `kubefirst clean`
	// if !viper.GetBool("template-repo.gitops.cloned") || viper.GetBool("template-repo.gitops.removed") {

	progressPrinter.IncrementTracker("preflight-checks", 1)

	// prepare repos
	progressPrinter.AddTracker("cloning-and-formatting-git-repositories", "Cloning and formatting git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.gitops-ready-to-push") {
		if err := o.SetupRepos(config); err != nil {
			return fmt.Errorf("error setting up repositories: %s", err.Error())
		}
	}

	log.Info().Msg("completed gitops repo generation - continuing")

	progressPrinter.IncrementTracker("cloning-and-formatting-git-repositories", 1)

	// run tf apply
	progressPrinter.AddTracker("applying-git-terraform", fmt.Sprintf("Applying %s Terraform", config.GitProvider), 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	tfEnvs := map[string]string{
		"TF_VAR_kbot_ssh_public_key":   viper.GetString("kbot.public-key"),
		"TF_CLI_ARGS":                  "-no-color", // no color in tf output
		"AWS_ACCESS_KEY_ID":            pkg.MinioDefaultUsername,
		"AWS_SECRET_ACCESS_KEY":        pkg.MinioDefaultPassword,
		"TF_VAR_aws_access_key_id":     pkg.MinioDefaultUsername,
		"TF_VAR_aws_secret_access_key": pkg.MinioDefaultPassword,
	}

	if strings.ToLower(o.gitProvider) == "github" {
		tfEnvs["GITHUB_TOKEN"] = o.gitToken
		tfEnvs["GITHUB_OWNER"] = o.gitOwner
	}

	if strings.ToLower(o.gitProvider) == "gitlab" {
		tfEnvs["GITLAB_TOKEN"] = o.gitToken
		tfEnvs["GITLAB_OWNER"] = o.gitOwner
		tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(o.gitlabGroupId)
	}

	// apply terraform to create teams and repositories on the specified provider
	if !viper.GetBool(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", o.gitProvider)) {
		if err := o.terraformApply(tfEnvs, config); err != nil {
			return fmt.Errorf("error applying terraform: %s", err.Error())
		}
	}

	// push detokenized gitops-template repository content to new remote
	progressPrinter.AddTracker("pushing-gitops-repos-upstream", "Pushing git repositories", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msgf("referencing gitops repository: %s", config.DestinationGitopsRepoGitURL)
	log.Info().Msgf("referencing metaphor repository: %s", config.DestinationMetaphorRepoURL)

	// push gitops and metaphor repo
	if !viper.GetBool("kubefirst-checks.gitops-repo-pushed") {
		if err := o.PushRepo(config.GitopsDir, config); err != nil {
			return fmt.Errorf("error pushing gitops repo: %s", err.Error())
		}

		if err := o.PushRepo(config.MetaphorDir, config); err != nil {
			return fmt.Errorf("error pushing metaphor repo: %s", err.Error())
		}
	}

	log.Info().Msg("pushed detokenized gitops repository content")

	progressPrinter.IncrementTracker("pushing-gitops-repos-upstream", 1)

	// create k3d cluster
	progressPrinter.AddTracker("creating-k3d-cluster", "Creating k3d cluster", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		if err := o.createK3dCluster(config); err != nil {
			return fmt.Errorf("error creating k3d cluster: %s", err.Error())
		}

		return nil
	}

	log.Info().Msg("k3d cluster created")
	progressPrinter.IncrementTracker("creating-k3d-cluster", 1)

	// create k8s resoures
	progressPrinter.AddTracker("bootstrapping-kubernetes-resources", "Bootstrapping Kubernetes resources", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	if !viper.GetBool("kubefirst-checks.k8s-secrets-created") {
		if err := o.createK3dAtlantisWebhookSecret(kcfg, config); err != nil {
			return fmt.Errorf("error creating k3d secrets: %s", err.Error())
		}
	}

	log.Info().Msg("added secrets to k3d cluster")

	progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)

	// //* check for ssl restore
	// log.Info().Msg("checking for tls secrets to restore")
	// secretsFilesToRestore, err := ioutil.ReadDir(config.SSLBackupDir + "/secrets")
	// if err != nil {
	// 	log.Info().Msgf("%s", err)
	// }
	// if len(secretsFilesToRestore) != 0 {
	// 	// todo would like these but requires CRD's and is not currently supported
	// 	// add crds ( use execShellReturnErrors? )
	// 	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-clusterissuers.yaml
	// 	// https://raw.githubusercontent.com/cert-manager/cert-manager/v1.11.0/deploy/crds/crd-certificates.yaml
	// 	// add certificates, and clusterissuers
	// 	log.Info().Msgf("found %d tls secrets to restore", len(secretsFilesToRestore))
	// 	ssl.Restore(config.SSLBackupDir, k3d.DomainName, config.Kubeconfig)
	// } else {
	// 	log.Info().Msg("no files found in secrets directory, continuing")
	// }

	progressPrinter.IncrementTracker("bootstrapping-kubernetes-resources", 1)
	progressPrinter.AddTracker("verifying-k3d-cluster-readiness", "Verifying Kubernetes cluster is ready", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	// traefik
	traefikDeployment, err := k8s.ReturnDeploymentObject(kcfg.Clientset, "app.kubernetes.io/name", "traefik", "kube-system", 240)
	if err != nil {
		log.Error().Msgf("error finding traefik deployment: %s", err.Error())

		return err
	}

	if _, err = k8s.WaitForDeploymentReady(kcfg.Clientset, traefikDeployment, 240); err != nil {
		log.Error().Msgf("error waiting for traefik deployment ready state: %s", err.Error())

		return err
	}

	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness", 1)

	// metrics-server
	metricsServerDeployment, err := k8s.ReturnDeploymentObject(kcfg.Clientset, "k8s-app", "metrics-server", "kube-system", 240)
	if err != nil {
		log.Error().Msgf("error finding metrics-server deployment: %s", err.Error())

		return err
	}

	if _, err = k8s.WaitForDeploymentReady(kcfg.Clientset, metricsServerDeployment, 240); err != nil {
		log.Error().Msgf("error waiting for metrics-server deployment ready state: %s", err.Error())

		return err
	}

	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness", 1)

	time.Sleep(time.Second * 20)

	progressPrinter.IncrementTracker("verifying-k3d-cluster-readiness", 1)

	// install argocd
	progressPrinter.AddTracker("installing-argo-cd", "Installing and configuring Argo CD", 3)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.argocd-install") {
		if err := o.installArgoCD(kcfg); err != nil {
			return fmt.Errorf("error installing ArgoCD: %s", err.Error())
		}
	}

	log.Info().Msg("argo cd installed - continuing")

	progressPrinter.IncrementTracker("installing-argo-cd", 1)

	// Wait for ArgoCD to be ready
	if _, err := k8s.VerifyArgoCDReadiness(kcfg.Clientset, true, 300); err != nil {
		log.Error().Msgf("error waiting for ArgoCD to become ready: %s", err.Error())

		return err
	}

	//* argocd pods are ready, get and set credentials
	if !viper.GetBool("kubefirst-checks.argocd-credentials-set") {
		if err := o.argoCDCredentials(kcfg); err != nil {
			return fmt.Errorf("error getting argocd credentials: %s", err.Error())
		}
	}

	log.Info().Msg("argo credentials set, continuing")

	progressPrinter.IncrementTracker("installing-argo-cd", 1)

	//* argocd sync registry and start sync waves
	if !viper.GetBool("kubefirst-checks.argocd-create-registry") {
		if err := o.argoCDRegistry(kcfg); err != nil {
			return fmt.Errorf("error setting up argocd regigstry: %s", err.Error())
		}
	}

	log.Info().Msg("argocd registry create done, continuing")

	progressPrinter.IncrementTracker("installing-argo-cd", 1)

	// Wait for Vault StatefulSet Pods to transition to Running
	progressPrinter.AddTracker("configuring-vault", "Configuring Vault", 4)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(kcfg.Clientset, "app.kubernetes.io/instance", "vault", "vault", 120)
	if err != nil {
		log.Error().Msgf("Error finding Vault StatefulSet: %s", err.Error())

		return err
	}

	if _, err := k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 120, true); err != nil {
		log.Error().Msgf("Error waiting for Vault StatefulSet ready state: %s", err.Error())

		return err
	}

	progressPrinter.IncrementTracker("configuring-vault", 1)

	// Init and unseal vault
	// We need to wait before we try to run any of these commands or there may be
	// unexpected timeouts
	time.Sleep(time.Second * 10)

	progressPrinter.IncrementTracker("configuring-vault", 1)

	if !viper.GetBool("kubefirst-checks.vault-initialized") {
		if err := o.vaultInitialize(kcfg); err != nil {
			return fmt.Errorf("error initializing Vault: %s", err.Error())
		}
	}

	log.Info().Msg("vault is initialized")

	progressPrinter.IncrementTracker("configuring-vault", 1)

	if err := o.setupMinio(kcfg, config); err != nil {
		return fmt.Errorf("error setting up minio: %s", err.Error())
	}

	// configure vault with terraform
	progressPrinter.IncrementTracker("configuring-vault", 1)

	var vaultRootToken string

	if !viper.GetBool("kubefirst-checks.terraform-apply-vault") {
		vaultRootToken, err = o.vaultConfigure(kcfg, config)
		if err != nil {
			return fmt.Errorf("error configuring Vault with terraform: %s", err.Error())
		}
	}

	log.Info().Msg("executed vault terraform")

	progressPrinter.IncrementTracker("configuring-vault", 1)

	// creating users
	progressPrinter.AddTracker("creating-users", "Creating users", 1)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if !viper.GetBool("kubefirst-checks.terraform-apply-users") {
		if err := o.vaultUsers(vaultRootToken, config); err != nil {
			return fmt.Errorf("error creating users: %s", err.Error())
		}
	}

	log.Info().Msg("created users with terraform")

	progressPrinter.IncrementTracker("creating-users", 1)

	// wrap up
	progressPrinter.AddTracker("wrapping-up", "Wrapping up", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	if err := o.wrapUp(config); err != nil {
		return fmt.Errorf("error commiting dekonzied files: %s", err.Error())
	}

	progressPrinter.IncrementTracker("wrapping-up", 1)

	// Wait for console Deployment Pods to transition to Running
	argoDeployment, err := k8s.ReturnDeploymentObject(kcfg.Clientset, "app.kubernetes.io/instance", "argo", "argo", 1200)
	if err != nil {
		log.Error().Msgf("Error finding argo workflows Deployment: %s", err.Error())

		return err
	}

	if _, err = k8s.WaitForDeploymentReady(kcfg.Clientset, argoDeployment, 120); err != nil {
		log.Error().Msgf("Error waiting for argo workflows Deployment ready state: %s", err.Error())

		return err
	}

	// Set flags used to track status of active options
	helpers.SetClusterStatusFlags(k3d.CloudProvider, config.GitProvider)

	cluster := utilities.CreateClusterRecordFromRaw(o.useTelemetry, o.gitOwner, o.gitUser, o.gitToken, o.gitlabGroupId, o.gitopsTemplateURL, o.gitopsTemplateBranch, o.catalogApps)

	if err := utilities.ExportCluster(cluster, kcfg); err != nil {
		log.Error().Err(err).Msg("error exporting cluster object")

		viper.Set("kubefirst.setup-complete", false)
		viper.Set("kubefirst-checks.cluster-install-complete", false)

		if err := viper.WriteConfig(); err != nil {
			log.Fatal().Msgf("cannot save state: %s", err.Error())
		}

		return err
	}

	kubefirstDeployment, err := k8s.ReturnDeploymentObject(kcfg.Clientset, "app.kubernetes.io/instance", "kubefirst", "kubefirst", 600)
	if err != nil {
		log.Error().Msgf("Error finding kubefirst Deployment: %s", err.Error())

		return err
	}

	if _, err = k8s.WaitForDeploymentReady(kcfg.Clientset, kubefirstDeployment, 120); err != nil {
		log.Error().Msgf("Error waiting for kubefirst Deployment ready state: %s", err.Error())

		return err
	}

	progressPrinter.IncrementTracker("wrapping-up", 1)

	if pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLTLS); err != nil {
		log.Error().Err(err).Msg("")
	}

	// Mark cluster install as complete
	if e := telemetry.SendEvent(o.segClient, telemetry.ClusterInstallCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	viper.Set("kubefirst-checks.cluster-install-complete", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	log.Info().Msg("kubefirst installation complete")
	log.Info().Msg("welcome to your new kubefirst platform running in K3d")

	time.Sleep(time.Second * 1) // allows progress bars to finish

	reports.LocalHandoffScreenV2(viper.GetString("components.argocd.password"), o.clusterName, o.gitDescriptor, o.gitOwner, config, o.ci)

	if o.ci {
		os.Exit(0)
	}

	return nil
}

func (o *createOptions) githubAuth() error {
	o.gitHost = k3d.GithubHost
	o.containerRegistryHost = "ghcr.io"

	// Attempt to retrieve session-scoped token for GitHub user
	gitHubService := services.NewGitHubService(o.httpClient)
	gitHubHandler := handlers.NewGitHubHandler(gitHubService)

	ghToken := utilities.EnvOrDefault("GITHUB_TOKEN", viper.GetString("github.session_token"))
	gitHubAccessToken, err := wrappers.AuthenticateGitHubUserWrapper(ghToken, gitHubHandler)
	if err != nil {
		log.Warn().Msgf(err.Error())
	}

	o.gitToken = gitHubAccessToken

	if err := github.VerifyTokenPermissions(o.gitToken); err != nil {
		return err
	}

	log.Info().Msg("verifying github authentication")

	githubUser, err := gitHubHandler.GetGitHubUser(o.gitToken)
	if err != nil {
		return err
	}

	// Owner is either an organization or a personal user's GitHub handle
	if o.githubOrg != "" {
		o.gitOwner = o.githubOrg
		o.gitDescriptor = "Organization"
	} else if o.githubUser != "" {
		o.gitOwner = githubUser
		o.gitDescriptor = "User"
	} else if o.githubOrg == "" && o.githubUser == "" {
		o.gitOwner = githubUser
	}

	o.githubUser = githubUser
	o.gitUser = o.githubUser

	viper.Set("flags.github-owner", o.gitOwner)
	viper.Set("github.session_token", o.gitToken)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	return nil
}

func (o *createOptions) gitlabAuth() error {
	o.gitToken = os.Getenv("GITLAB_TOKEN")

	// Verify token scopes
	if err := gitlab.VerifyTokenPermissions(o.gitToken); err != nil {
		return err
	}

	gitlabClient, err := gitlab.NewGitLabClient(o.gitToken, o.gitlabGroup)
	if err != nil {
		return err
	}

	o.gitHost = k3d.GitlabHost
	o.gitOwner = gitlabClient.ParentGroupPath
	o.gitlabGroupId = gitlabClient.ParentGroupID
	o.gitDescriptor = "Group"

	log.Info().Msgf("set gitlab owner to %s", o.gitOwner)

	// Get authenticated user's name
	user, _, err := gitlabClient.Client.Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
	}

	o.gitUser = user.Username
	o.containerRegistryHost = "registry.gitlab.com"

	viper.Set("flags.gitlab-owner", o.gitlabGroup)
	viper.Set("flags.gitlab-owner-group-id", o.gitlabGroupId)
	viper.Set("gitlab.session_token", o.gitToken)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	return nil
}

func (o *createOptions) CheckGitCredentials(config *k3d.K3dConfig) error {
	newRepositoryNames := []string{"gitops", "metaphor"}
	newTeamNames := []string{"admins", "developers"}

	if e := telemetry.SendEvent(o.segClient, telemetry.GitCredentialsCheckStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	if len(o.gitToken) == 0 {
		msg := fmt.Sprintf("please set a %s_TOKEN environment variable to continue", strings.ToUpper(config.GitProvider))

		if e := telemetry.SendEvent(o.segClient, telemetry.GitCredentialsCheckFailed, msg); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}
	}

	initGitParameters := gitShim.GitInitParameters{
		GitProvider:  o.gitProvider,
		GitToken:     o.gitToken,
		GitOwner:     o.gitOwner,
		Repositories: newRepositoryNames,
		Teams:        newTeamNames,
	}

	if err := gitShim.InitializeGitProvider(&initGitParameters); err != nil {
		return err
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", config.GitProvider), true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.GitCredentialsCheckCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) setupKbot() error {
	if e := telemetry.SendEvent(o.segClient, telemetry.KbotSetupStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")

	sshPrivateKey, sshPublicKey, err := utils.CreateSshKeyPair()
	if err != nil {
		log.Warn().Msgf("generate public keys failed: %s\n", err.Error())

		if e := telemetry.SendEvent(o.segClient, telemetry.KbotSetupFailed, err.Error()); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}
	}

	log.Info().Msg("ssh key pair creation complete")

	viper.Set("kbot.private-key", sshPrivateKey)
	viper.Set("kbot.public-key", sshPublicKey)
	viper.Set("kbot.username", "kbot")
	viper.Set("kubefirst-checks.kbot-setup", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.KbotSetupCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) downnloadTools(config *k3d.K3dConfig) error {
	log.Info().Msg("installing kubefirst dependencies")

	if err := k3d.DownloadTools(o.clusterName, config.GitProvider, o.gitOwner, config.ToolsDir, config.GitProtocol); err != nil {
		return err
	}

	log.Info().Msg("download dependencies `$HOME/.k1/tools` complete")

	viper.Set("kubefirst-checks.tools-downloaded", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	return nil
}

func (o *createOptions) terraformApply(tfEnvs map[string]string, config *k3d.K3dConfig) error {
	if e := telemetry.SendEvent(o.segClient, telemetry.GitTerraformApplyStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	log.Info().Msgf("Creating %s resources with Terraform", o.gitProvider)

	tfEntrypoint := path.Join(config.GitopsDir, "terraform", o.gitProvider)

	// Erase public key to prevent it from being created if the git protocol argument is set to htps
	if config.GitProtocol == "https" {
		tfEnvs["TF_VAR_kbot_ssh_public_key"] = ""
	}

	if err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs); err != nil {
		if e := telemetry.SendEvent(o.segClient, telemetry.GitTerraformApplyFailed, fmt.Sprintf("error creating %s resources with terraform %s: %s", o.gitProvider, tfEntrypoint, err)); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}
	}

	log.Info().Msgf("created git repositories for %s.com/%s", o.gitProvider, config.MetaphorDir)
	log.Info().Msgf("created git repositories for %s.com/%s", o.gitProvider, config.GitopsDir)

	viper.Set(fmt.Sprintf("kubefirst-checks.terraform-apply-%s", o.gitProvider), true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.GitTerraformApplyCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) k3dGitOpsDirectoryValues(config *k3d.K3dConfig) *k3d.GitopsDirectoryValues {
	return &k3d.GitopsDirectoryValues{
		GithubOwner:                   o.gitOwner,
		GithubUser:                    o.gitUser,
		GitlabOwner:                   o.gitOwner,
		GitlabOwnerGroupID:            o.gitlabGroupId,
		GitlabUser:                    o.gitUser,
		DomainName:                    k3d.DomainName,
		AtlantisAllowList:             fmt.Sprintf("%s/%s/*", o.gitHost, o.gitOwner),
		AlertsEmail:                   "REMOVE_THIS_VALUE",
		ClusterName:                   o.clusterName,
		ClusterType:                   o.clusterType,
		GithubHost:                    k3d.GithubHost,
		GitlabHost:                    k3d.GitlabHost,
		ArgoWorkflowsIngressURL:       fmt.Sprintf("https://argo.%s", k3d.DomainName),
		VaultIngressURL:               fmt.Sprintf("https://vault.%s", k3d.DomainName),
		ArgocdIngressURL:              fmt.Sprintf("https://argocd.%s", k3d.DomainName),
		AtlantisIngressURL:            fmt.Sprintf("https://atlantis.%s", k3d.DomainName),
		MetaphorDevelopmentIngressURL: fmt.Sprintf("https://metaphor-development.%s", k3d.DomainName),
		MetaphorStagingIngressURL:     fmt.Sprintf("https://metaphor-staging.%s", k3d.DomainName),
		MetaphorProductionIngressURL:  fmt.Sprintf("https://metaphor-production.%s", k3d.DomainName),
		KubefirstVersion:              configs.K1Version,
		KubefirstTeam:                 utilities.EnvOrDefault("KUBEFIRST_TEAM", "false"),
		KubeconfigPath:                config.Kubeconfig,
		GitopsRepoURL:                 o.gitopsRepoURL,
		GitProvider:                   config.GitProvider,
		ClusterId:                     o.clusterId,
		CloudProvider:                 k3d.CloudProvider,
		UseTelemetry:                  fmt.Sprintf("%v", o.useTelemetry),
	}
}

func (o *createOptions) k3dMetaphorTokenValues() *k3d.MetaphorTokenValues {
	return &k3d.MetaphorTokenValues{
		ClusterName:                   o.clusterName,
		CloudRegion:                   o.cloudRegion,
		ContainerRegistryURL:          fmt.Sprintf("%s/%s/metaphor", o.containerRegistryHost, o.gitOwner),
		DomainName:                    k3d.DomainName,
		MetaphorDevelopmentIngressURL: fmt.Sprintf("metaphor-development.%s", k3d.DomainName),
		MetaphorStagingIngressURL:     fmt.Sprintf("metaphor-staging.%s", k3d.DomainName),
		MetaphorProductionIngressURL:  fmt.Sprintf("metaphor-production.%s", k3d.DomainName),
	}
}

func (o *createOptions) SetupRepos(config *k3d.K3dConfig) error {
	var removeAtlantis bool

	log.Info().Msg("generating your new gitops repository")

	if viper.GetString("secrets.atlantis-ngrok-authtoken") == "" {
		removeAtlantis = true
	}

	if err := k3d.PrepareGitRepositories(
		config.GitProvider,
		o.clusterName,
		o.clusterType,
		config.DestinationGitopsRepoURL, // default to https for git interactions when creating remotes
		config.GitopsDir,
		o.gitopsTemplateBranch,
		o.gitopsTemplateURL,
		config.DestinationMetaphorRepoURL, // default to https for git interactions when creating remotes
		config.K1Dir,
		o.k3dGitOpsDirectoryValues(config),
		config.MetaphorDir,
		o.k3dMetaphorTokenValues(),
		o.gitProtocol,
		removeAtlantis,
	); err != nil {
		return err
	}

	// todo emit init telemetry end
	viper.Set("kubefirst-checks.gitops-ready-to-push", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	return nil
}

func (o *createOptions) PushRepo(repo string, config *k3d.K3dConfig) error {
	httpAuth := &githttps.BasicAuth{
		Username: o.gitUser,
		Password: o.gitToken,
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.GitopsRepoPushStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	r, err := git.PlainOpen(repo)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", repo)
	}

	if strings.ToLower(o.gitProvider) == "gitlab" {
		if err := utils.EvalSSHKey(&types.EvalSSHKeyRequest{
			GitProvider:     o.gitProvider,
			GitlabGroupFlag: o.gitlabGroup,
			GitToken:        o.gitToken,
		}); err != nil {
			return err
		}
	}

	if err := r.Push(&git.PushOptions{
		RemoteName: config.GitProvider,
		Auth:       httpAuth,
	}); err != nil {
		msg := fmt.Sprintf("error pushing detokenized gitops repository to remote %s: %s", config.DestinationGitopsRepoGitURL, err)

		if e := telemetry.SendEvent(o.segClient, telemetry.GitopsRepoPushFailed, msg); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}

		if !strings.Contains(msg, "already up-to-date") {
			log.Panic().Msg(msg)
		}
	}

	log.Info().Msgf("successfully pushed %s repository to https://%s/%s", repo, o.gitHost, o.gitOwner)

	// todo delete the local gitops repo and re-clone it
	// todo that way we can stop worrying about which origin we're going to push to
	viper.Set("kubefirst-checks.gitops-repo-pushed", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.GitopsRepoPushCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) createK3dCluster(config *k3d.K3dConfig) error {
	log.Info().Msg("Creating k3d cluster")

	if e := telemetry.SendEvent(o.segClient, telemetry.CloudTerraformApplyStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	if err := k3d.ClusterCreate(o.clusterName, config.K1Dir, config.K3dClient, config.Kubeconfig); err != nil {
		msg := fmt.Sprintf("error creating k3d resources with k3d client %s: %s", config.K3dClient, err)

		viper.Set("kubefirst-checks.create-k3d-cluster-failed", true)

		if err := viper.WriteConfig(); err != nil {
			log.Fatal().Msgf("cannot save state: %s", err.Error())
		}

		if e := telemetry.SendEvent(o.segClient, telemetry.CloudTerraformApplyFailed, msg); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}

		return fmt.Errorf(msg)
	}

	viper.Set("kubefirst-checks.create-k3d-cluster", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.CloudTerraformApplyCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) createK3dAtlantisWebhookSecret(kcfg *k8s.KubernetesClient, config *k3d.K3dConfig) error {
	if err := k3d.GenerateTLSSecrets(kcfg.Clientset, *config); err != nil {
		return err
	}
	// gen & store atlantis secret if none exists
	atlantisWebhookSecret := viper.GetString("secrets.atlantis-webhook")
	if atlantisWebhookSecret == "" {
		atlantisWebhookSecret = pkg.Random(20)
		viper.Set("secrets.atlantis-webhook", atlantisWebhookSecret)

		if err := viper.WriteConfig(); err != nil {
			log.Fatal().Msgf("cannot save state: %s", err.Error())
		}
	}

	if err := k3d.AddK3DSecrets(
		atlantisWebhookSecret,
		viper.GetString("kbot.public-key"),
		o.gitopsRepoURL,
		viper.GetString("kbot.private-key"),
		config.GitProvider,
		o.gitUser,
		o.gitOwner,
		config.Kubeconfig,
		o.gitToken,
	); err != nil {
		log.Info().Msg("Error adding kubernetes secrets for bootstrap")

		return err
	}

	viper.Set("kubefirst-checks.k8s-secrets-created", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	return nil
}

func (o *createOptions) installArgoCD(kcfg *k8s.KubernetesClient) error {
	argoCDInstallPath := fmt.Sprintf("github.com:kubefirst/manifests/argocd/k3d?ref=%s", pkg.KubefirstManifestRepoRef)

	if e := telemetry.SendEvent(o.segClient, telemetry.ArgoCDInstallStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	log.Info().Msgf("installing argocd")

	// Build and apply manifests
	yamlData, err := kcfg.KustomizeBuild(argoCDInstallPath)
	if err != nil {
		return err
	}

	output, err := kcfg.SplitYAMLFile(yamlData)
	if err != nil {
		return err
	}

	if kcfg.ApplyObjects("", output) != nil {
		if e := telemetry.SendEvent(o.segClient, telemetry.ArgoCDInstallFailed, err.Error()); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}

		return err
	}

	viper.Set("kubefirst-checks.argocd-install", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.ArgoCDInstallCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) argoCDCredentials(kcfg *k8s.KubernetesClient) error {
	var argocdPassword, argoCDToken string

	log.Info().Msg("Setting argocd username and password credentials")

	argocd.ArgocdSecretClient = kcfg.Clientset.CoreV1().Secrets("argocd")

	argocdPassword = k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
	if argocdPassword == "" {
		log.Info().Msg("argocd password not found in secret")

		return fmt.Errorf("could not find argocd secret")
	}

	viper.Set("components.argocd.password", argocdPassword)
	viper.Set("components.argocd.username", "admin")

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	log.Info().Msg("argocd username and password credentials set successfully")
	log.Info().Msg("Getting an argocd auth token")

	// only the host, not the protocol
	argoCDURL := k3d.ArgocdURL

	if helpers.TestEndpointTLS(strings.Replace(k3d.ArgocdURL, "https://", "", 1)) != nil {
		argoCDStopChannel := make(chan struct{}, 1)

		log.Info().Msgf("argocd not available via https, using http")

		defer func() {
			close(argoCDStopChannel)
		}()

		k8s.OpenPortForwardPodWrapper(kcfg.Clientset, kcfg.RestConfig, "argocd-server", "argocd", 8080, 8080, argoCDStopChannel)

		argoCDURL = strings.Replace(k3d.ArgocdURL, "https://", "http://", 1) + ":8080"
	}

	argoCDToken, err := argocd.GetArgocdTokenV2(o.httpClient, argoCDURL, "admin", argocdPassword)
	if err != nil {
		return err
	}

	log.Info().Msg("argocd admin auth token set")

	viper.Set("components.argocd.auth-token", argoCDToken)
	viper.Set("kubefirst-checks.argocd-credentials-set", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if configs.K1Version == "development" {
		if err := clipboard.WriteAll(argocdPassword); err != nil {
			log.Error().Err(err).Msg("error copying argocd password to clipboard")
		}

		if _, ok := os.LookupEnv("SKIP_ARGOCD_LAUNCH"); !ok || !o.ci {
			if pkg.OpenBrowser(pkg.ArgoCDLocalURLTLS) != nil {
				log.Error().Err(err).Msg("error opening argocd in browser")
			}
		}
	}

	return nil
}

func (o *createOptions) argoCDRegistry(kcfg *k8s.KubernetesClient) error {
	if e := telemetry.SendEvent(o.segClient, telemetry.CreateRegistryStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
	if err != nil {
		return err
	}

	log.Info().Msg("applying the registry application to argocd")

	registryApplicationObject := argocd.GetArgoCDApplicationObject(o.gitopsRepoURL, fmt.Sprintf("registry/%s", o.clusterName))

	_, _ = argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})

	viper.Set("kubefirst-checks.argocd-create-registry", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.CreateRegistryCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) vaultInitialize(kcfg *k8s.KubernetesClient) error {
	if e := telemetry.SendEvent(o.segClient, telemetry.VaultInitializationStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	// Initialize and unseal Vault
	vaultHandlerPath := "github.com:kubefirst/manifests.git/vault-handler/replicas-1"

	// Build and apply manifests
	yamlData, err := kcfg.KustomizeBuild(vaultHandlerPath)
	if err != nil {
		return err
	}

	output, err := kcfg.SplitYAMLFile(yamlData)
	if err != nil {
		return err
	}

	if err := kcfg.ApplyObjects("", output); err != nil {
		return err
	}

	// Wait for the Job to finish
	job, err := k8s.ReturnJobObject(kcfg.Clientset, "vault", "vault-handler")
	if err != nil {
		return err
	}

	_, err = k8s.WaitForJobComplete(kcfg.Clientset, job, 240)
	if err != nil {
		msg := fmt.Sprintf("could not run vault unseal job: %s", err.Error())

		if e := telemetry.SendEvent(o.segClient, telemetry.VaultInitializationFailed, msg); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}

		log.Fatal().Msg(msg)
	}

	viper.Set("kubefirst-checks.vault-initialized", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.VaultInitializationCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) containerRegistryAuth(kcfg *k8s.KubernetesClient) (string, error) {
	containerRegistryAuth := gitShim.ContainerRegistryAuth{
		GitProvider:           o.gitProvider,
		GitUser:               o.gitUser,
		GitToken:              o.gitToken,
		GitlabGroupFlag:       o.gitlabGroup,
		GithubOwner:           o.gitOwner,
		ContainerRegistryHost: o.containerRegistryHost,
		Clientset:             kcfg.Clientset,
	}

	containerRegistryAuthToken, err := gitShim.CreateContainerRegistrySecret(&containerRegistryAuth)
	if err != nil {
		return "", err
	}

	return containerRegistryAuthToken, nil
}

func (o *createOptions) vaultConfigure(kcfg *k8s.KubernetesClient, config *k3d.K3dConfig) (string, error) {
	tfEnvs := map[string]string{}
	var usernamePasswordString, base64DockerAuth string
	var vaultRootToken string

	// get & store ngrok token
	atlantisNgrokAuthtoken := viper.GetString("secrets.atlantis-ngrok-authtoken")
	if atlantisNgrokAuthtoken == "" {
		atlantisNgrokAuthtoken = os.Getenv("NGROK_AUTHTOKEN")
		viper.Set("secrets.atlantis-ngrok-authtoken", atlantisNgrokAuthtoken)

		if err := viper.WriteConfig(); err != nil {
			log.Fatal().Msgf("cannot save state: %s", err.Error())
		}
	}

	// port forward
	vaultStopChannel := make(chan struct{}, 1)

	defer func() {
		close(vaultStopChannel)
	}()

	k8s.OpenPortForwardPodWrapper(kcfg.Clientset, kcfg.RestConfig, "vault-0", "vault", 8200, 8200, vaultStopChannel)

	// container registry auth
	registryAuthToken, err := o.containerRegistryAuth(kcfg)
	if err != nil {
		return "", fmt.Errorf("error authentication to container registry: %s", err.Error())
	}

	// Retrieve root token from init step
	secData, err := k8s.ReadSecretV2(kcfg.Clientset, "vault", "vault-unseal-secret")
	if err != nil {
		return "", err
	}

	vaultRootToken = secData["root-token"]

	// Parse k3d api endpoint from kubeconfig
	// In this case, we need to get the IP of the in-cluster API server to provide to Vault
	// to work with Kubernetes auth
	kubernetesInClusterAPIService, err := k8s.ReadService(config.Kubeconfig, "default", "kubernetes")
	if err != nil {
		log.Error().Msgf("error looking up kubernetes api server service: %s", err.Error())

		return "", err
	}

	if err := helpers.TestEndpointTLS(strings.Replace(k3d.VaultURL, "https://", "", 1)); err != nil {
		return "", fmt.Errorf("unable to reach vault over https - this is likely due to the mkcert certificate store missing. please install it via `%s -install`", config.MkCertClient)
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.VaultTerraformApplyStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	usernamePasswordString = fmt.Sprintf("%s:%s", o.gitUser, o.gitToken)
	base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

	if config.GitProvider == "gitlab" {
		usernamePasswordString = fmt.Sprintf("%s:%s", "container-registry-auth", registryAuthToken)
		base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

		tfEnvs["TF_VAR_container_registry_auth"] = registryAuthToken
		tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(o.gitlabGroupId)
	}

	log.Info().Msg("configuring vault with terraform")

	tfEnvs["TF_CLI_ARGS"] = "-no-color" // no color in tf output
	tfEnvs["TF_VAR_email_address"] = "your@email.com"
	tfEnvs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = o.gitToken
	tfEnvs[fmt.Sprintf("TF_VAR_%s_user", config.GitProvider)] = o.gitUser
	tfEnvs["TF_VAR_vault_addr"] = k3d.VaultPortForwardURL
	tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth
	tfEnvs["TF_VAR_vault_token"] = vaultRootToken
	tfEnvs["VAULT_ADDR"] = k3d.VaultPortForwardURL
	tfEnvs["VAULT_TOKEN"] = vaultRootToken
	tfEnvs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	tfEnvs["TF_VAR_kbot_ssh_private_key"] = viper.GetString("kbot.private-key")
	tfEnvs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
	tfEnvs["TF_VAR_kubernetes_api_endpoint"] = fmt.Sprintf("https://%s", kubernetesInClusterAPIService.Spec.ClusterIP)
	tfEnvs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = viper.GetString(fmt.Sprintf("flags.%s-owner", config.GitProvider))
	tfEnvs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
	tfEnvs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
	tfEnvs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
	tfEnvs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword
	tfEnvs["TF_VAR_ngrok_authtoken"] = viper.GetString("secrets.atlantis-ngrok-authtoken")
	// tfEnvs["TF_LOG"] = "DEBUG"

	tfEntrypoint := config.GitopsDir + "/terraform/vault"
	if err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs); err != nil {
		if e := telemetry.SendEvent(o.segClient, telemetry.VaultTerraformApplyStarted, err.Error()); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}

		return "", err
	}

	log.Info().Msg("vault terraform executed successfully")

	viper.Set("kubefirst-checks.terraform-apply-vault", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.VaultTerraformApplyCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return vaultRootToken, nil
}

func (o *createOptions) vaultUsers(vaultRootToken string, config *k3d.K3dConfig) error {
	if e := telemetry.SendEvent(o.segClient, telemetry.UsersTerraformApplyStarted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	log.Info().Msg("applying users terraform")

	tfEnvs := map[string]string{
		"TF_VAR_email_address": "your@email.com",
		"TF_CLI_ARGS":          "-no-color", // no color in tf output
		fmt.Sprintf("TF_VAR_%s_token", config.GitProvider): o.gitToken,
		"TF_VAR_vault_addr":  k3d.VaultPortForwardURL,
		"TF_VAR_vault_token": vaultRootToken,
		"VAULT_ADDR":         k3d.VaultPortForwardURL,
		"VAULT_TOKEN":        vaultRootToken,
		fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider)): o.gitToken,
		fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider)): o.gitOwner,
	}

	tfEntrypoint := config.GitopsDir + "/terraform/users"
	if err := terraform.InitApplyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs); err != nil {
		if e := telemetry.SendEvent(o.segClient, telemetry.UsersTerraformApplyStarted, err.Error()); e != nil {
			log.Warn().Msgf("error sending telemetry event: %s", e.Error())
		}

		return err
	}

	viper.Set("kubefirst-checks.terraform-apply-users", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	if e := telemetry.SendEvent(o.segClient, telemetry.UsersTerraformApplyCompleted, ""); e != nil {
		log.Warn().Msgf("error sending telemetry event: %s", e.Error())
	}

	return nil
}

func (o *createOptions) setupMinio(kcfg *k8s.KubernetesClient, config *k3d.K3dConfig) error {
	minioStopChannel := make(chan struct{}, 1)

	ctx, _ := context.WithCancel(context.Background())

	defer func() {
		close(minioStopChannel)
	}()

	k8s.OpenPortForwardPodWrapper(kcfg.Clientset, kcfg.RestConfig, "minio", "minio", 9000, 9000, minioStopChannel)

	// Initialize minio client object.
	minioClient, err := minio.New(pkg.MinioPortForwardEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(pkg.MinioDefaultUsername, pkg.MinioDefaultPassword, ""),
		Secure: false,
		Region: pkg.MinioRegion,
	})
	if err != nil {
		log.Info().Msgf("Error creating Minio client: %s", err.Error())
	}

	// define upload object
	objectName := fmt.Sprintf("terraform/%s/terraform.tfstate", config.GitProvider)
	filePath := config.K1Dir + fmt.Sprintf("/gitops/%s", objectName)
	contentType := "xl.meta"
	bucketName := "kubefirst-state-store"

	log.Info().Msgf("BucketName: %s", bucketName)

	viper.Set("kubefirst.state-store.name", bucketName)
	viper.Set("kubefirst.state-store.hostname", "minio-console.kubefirst.dev")
	viper.Set("kubefirst.state-store-creds.access-key-id", pkg.MinioDefaultUsername)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", pkg.MinioDefaultPassword)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	// Upload the zip file with FPutObject
	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		log.Info().Msgf("Error uploading to Minio bucket: %s", err.Error())
	}

	log.Printf("Successfully uploaded %s to bucket %s\n", objectName, info.Bucket)

	return nil
}

func (o *createOptions) wrapUp(config *k3d.K3dConfig) error {
	httpAuth := &githttps.BasicAuth{
		Username: o.gitUser,
		Password: o.gitToken,
	}

	if err := k3d.PostRunPrepareGitopsRepository(o.clusterName, config.GitopsDir, o.k3dGitOpsDirectoryValues(config)); err != nil {
		log.Info().Msgf("Error detokenize post run: %s", err.Error())
	}

	gitopsRepo, err := git.PlainOpen(config.GitopsDir)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", config.GitopsDir)
	}

	// check if file exists before rename
	if _, err := os.Stat(fmt.Sprintf("%s/terraform/%s/remote-backend.md", config.GitopsDir, config.GitProvider)); err == nil {
		if err := os.Rename(
			fmt.Sprintf("%s/terraform/%s/remote-backend.md", config.GitopsDir, config.GitProvider),
			fmt.Sprintf("%s/terraform/%s/remote-backend.tf", config.GitopsDir, config.GitProvider)); err != nil {
			return err
		}
	}

	viper.Set("kubefirst-checks.post-detokenize", true)

	if err := viper.WriteConfig(); err != nil {
		log.Fatal().Msgf("cannot save state: %s", err.Error())
	}

	// Final gitops repo commit and push
	if err := gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content post run"); err != nil {
		return err
	}

	if err := gitopsRepo.Push(&git.PushOptions{
		RemoteName: config.GitProvider,
		Auth:       httpAuth,
	}); err != nil {
		log.Info().Msgf("Error pushing repo: %s", err.Error())
	}

	return nil
}
