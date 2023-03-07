package local

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kubefirst/kubefirst/internal/ssh"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/addon"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func validateLocal(cmd *cobra.Command, args []string) error {

	// set log level
	log.Info().Msgf("setting log level to: %s", logLevel)
	zerologLevel := pkg.GetLogLevelByString(logLevel)
	zerolog.SetGlobalLevel(zerologLevel)

	config := configs.ReadConfig()

	// gitProvider := viper.GetString("git-provider")
	// cloud := viper.GetString("cloud")
	clusterId := uuid.New().String()

	// if useTelemetry {
	// 	pkg.InformUser("Sending installation telemetry", silentMode)
	// 	if err := wrappers.SendSegmentIoTelemetry("", pkg.MetricInitStarted, cloud, gitProvider, clusterId); err != nil {
	// 		log.Error().Err(err).Msg("")
	// 	}
	// }

	if err := pkg.ValidateK1Folder(config.K1FolderPath); err != nil {
		return err
	}

	// check disk
	free, err := pkg.GetAvailableDiskSize()
	if err != nil {
		return err
	}

	// convert available disk size to GB format
	availableDiskSize := float64(free) / humanize.GByte
	if availableDiskSize < pkg.MinimumAvailableDiskSize {
		return fmt.Errorf(
			"there is not enough space to proceed with the installation, a minimum of %d GB is required to proceed",
			pkg.MinimumAvailableDiskSize,
		)
	}

	// set default values to kubefirst file
	viper.Set("gitops.repo", pkg.KubefirstGitOpsRepository)
	viper.Set("gitops.owner", "kubefirst")
	viper.Set("git-provider", pkg.GitHubProviderName)
	viper.Set("metaphor.branch", metaphorBranch)

	viper.Set("gitops.branch", gitOpsBranch)
	viper.Set("github.owner", viper.GetString("github.user"))
	viper.Set("cloud", pkg.CloudK3d)
	viper.Set("cluster-name", pkg.LocalClusterName)
	viper.Set("adminemail", adminEmail)

	viper.Set("argocd.local.service", pkg.ArgoCDLocalURL)
	viper.Set("vault.local.service", pkg.VaultLocalURLTLS)
	viper.Set("use-telemetry", useTelemetry)
	viper.Set("cluster-id", clusterId)

	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	// addons
	addon.AddAddon("github")
	addon.AddAddon("k3d")
	// used for letsencrypt notifications and the gitlab root account
	if !skipMetaphor {
		addon.AddAddon("metaphor")
	}

	httpClient := http.DefaultClient
	gitHubService := services.NewGitHubService(httpClient)
	gitHubHandler := handlers.NewGitHubHandler(gitHubService)
	gitHubAccessToken, err := wrappers.AuthenticateGitHubUserWrapper(config, gitHubHandler)
	if err != nil {
		return err
	}

	// get GitHub data to set user and owner based on the provided token
	githubUser, err := gitHubHandler.GetGitHubUser(gitHubAccessToken)
	if err != nil {
		return err
	}

	// creates a new context, and a cancel function that allows canceling the context. The context is passed as an
	// argument to the RunNgrok function, which is then started in a new goroutine.
	var ctx context.Context
	ctx, cancelContext = context.WithCancel(context.Background())
	go pkg.RunNgrok(ctx)

	viper.Set("github.atlantis.webhook.secret", pkg.Random(20))
	viper.Set("github.user", githubUser)
	viper.Set("github.owner", githubUser)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	if silentMode {
		pkg.InformUser(
			"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
			silentMode,
		)
	}

	//progress bars are global, it is not needed to initialized on every stage.
	progressPrinter.SetupProgress(8, silentMode)

	progressPrinter.AddTracker("step-0", "Process Parameters", 1)
	progressPrinter.AddTracker("step-download", pkg.DownloadDependencies, 3)
	progressPrinter.AddTracker("step-gitops", pkg.CloneAndDetokenizeGitOpsTemplate, 1)
	progressPrinter.AddTracker("step-ssh", pkg.CreateSSHKey, 1)

	log.Info().Msg("installing kubefirst dependencies")

	progressPrinter.IncrementTracker("step-download", 1)
	err = downloadManager.DownloadTools(config)
	if err != nil {
		return err
	}
	log.Info().Msg("dependency installation complete")
	progressPrinter.IncrementTracker("step-download", 1)
	err = downloadManager.DownloadLocalTools(config)
	if err != nil {
		return err
	}

	progressPrinter.IncrementTracker("step-download", 1)

	log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
	ssh.CreateSshKeyPair()
	log.Info().Msg("ssh key pair creation complete")
	progressPrinter.IncrementTracker("step-ssh", 1)

	//
	// clone gitops template
	//
	// todo: temporary code, the full logic will be refactored in the next release

	// translation:
	//  - if not an execution from a released/binary kubefirst version / development version
	//  - and metaphor branch is not set, use the default branch
	if configs.K1Version == configs.DefaultK1Version && metaphorBranch == "" {
		metaphorBranch = "main"
	}

	if configs.K1Version == configs.DefaultK1Version {

		gitHubOrg := "kubefirst"
		repoName := "gitops"

		repoURL := fmt.Sprintf("https://github.com/%s/%s-template", gitHubOrg, repoName)
		branch := gitOpsBranch
		if gitOpsBranch == "" {
			//to fix, noldflags to default to main as default branch
			branch = "main"
		}
		_, err := gitClient.CloneBranchSetMain(repoURL, config.GitOpsLocalRepoPath, branch)
		if err != nil {
			return err
		}

		viper.Set("init.repos.gitops.cloned", true)
		viper.Set(fmt.Sprintf("git.clone.%s.branch", repoName), gitOpsBranch)
		if err = viper.WriteConfig(); err != nil {
			log.Error().Err(err).Msg("")
		}

	} else {
		//Tag should be used in absence of branch been provided
		//We should be able to change repo address and names from flags
		//The branch support is not meant only for developement mode, it is also for troubleshooting of releases, bug fixes.
		//Please, don't disable its support - even the binary from a release must support branch use.
		if gitOpsBranch != "" {
			repoURL := fmt.Sprintf("https://github.com/%s/%s-template", gitOpsOrg, gitOpsRepo)
			_, err := gitClient.CloneBranchSetMain(repoURL, config.GitOpsLocalRepoPath, gitOpsBranch)
			if err != nil {
				return err
			}
			viper.Set("init.repos.gitops.cloned", true)
			viper.Set(fmt.Sprintf("git.clone.%s.branch", gitOpsRepo), gitOpsBranch)
			if err = viper.WriteConfig(); err != nil {
				log.Error().Err(err).Msg("")
			}
		} else {
			// use tag
			gitHubOrg := "kubefirst"
			repoName := "gitops"

			tag := configs.K1Version
			_, err := gitClient.CloneTagSetMain(config.GitOpsLocalRepoPath, gitHubOrg, repoName, tag)
			if err != nil {
				return err
			}

			viper.Set(fmt.Sprintf("git.clone.%s.tag", repoName), tag)
			viper.Set("init.repos.gitops.cloned", true)
			if err = viper.WriteConfig(); err != nil {
				log.Error().Err(err).Msg("")
			}
		}
	}

	if !viper.GetBool("github.gitops.hydrated") {
		err = repo.UpdateForLocalMode(config.GitOpsLocalRepoPath)
		if err != nil {
			return err
		}
	}

	pkg.Detokenize(config.GitOpsLocalRepoPath)
	viper.Set(fmt.Sprintf("init.repos.%s.detokenized", pkg.KubefirstGitOpsRepository), true)
	if err = viper.WriteConfig(); err != nil {
		log.Error().Err(err).Msg("")
	}

	err = gitClient.CreateGitHubRemote(config.GitOpsLocalRepoPath, githubUser, pkg.KubefirstGitOpsRepository)
	if err != nil {
		return err
	}

	progressPrinter.IncrementTracker("step-gitops", 1)

	log.Info().Msg("sending init completed metric")

	pkg.InformUser("initialization step is done!", silentMode)

	// if useTelemetry {
	// 	if err = wrappers.SendSegmentIoTelemetry("", pkg.MetricInitCompleted, cloud, gitProvider, clusterId); err != nil {
	// 		log.Error().Err(err).Msg("")
	// 	}
	// }

	progressPrinter.IncrementTracker("step-0", 1)
	time.Sleep(100 * time.Millisecond) // necessary to wait progress bar to finish

	return nil
}

// validateDestroy validates primordial inputs before destroy command can be called.
func validateDestroy(cmd *cobra.Command, args []string) error {

	if silentMode {
		pkg.InformUser(
			"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
			silentMode,
		)
	}

	config := configs.ReadConfig()

	log.Info().Msg("setting GitHub token...")
	httpClient := http.DefaultClient
	gitHubService := services.NewGitHubService(httpClient)
	gitHubHandler := handlers.NewGitHubHandler(gitHubService)
	_, err := wrappers.AuthenticateGitHubUserWrapper(config, gitHubHandler)
	if err != nil {
		return err
	}
	log.Info().Msg("GitHub token set!")

	log.Info().Msg("updating Terraform backend for localhost instead of minio...")
	err = pkg.UpdateTerraformS3BackendForLocalhostAddress()
	if err != nil {
		return err
	}
	log.Info().Msg("updating Terraform backend for localhost instead of minio, done")

	return nil
}
