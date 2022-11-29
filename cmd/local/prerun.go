package local

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/addon"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
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
	"net/http"
	"time"
)

func validateLocal(cmd *cobra.Command, args []string) error {

	// set log level
	log.Info().Msgf("setting log level to: %s", logLevel)
	zerologLevel := pkg.GetLogLevelByString(logLevel)
	zerolog.SetGlobalLevel(zerologLevel)

	config := configs.ReadConfig()

	log.Info().Msg("sending init started metric")

	if useTelemetry {
		if err := wrappers.SendSegmentIoTelemetry("", pkg.MetricInitStarted); err != nil {
			log.Error().Err(err).Msg("")
		}
	}

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

	// if non-development/built/released version, set template tag version to clone tagged templates, in that way
	// the current built version, uses the same template version.
	// example: kubefirst version 1.10.3, has template repositories (gitops and metaphor's) tags set as 1.10.3
	// when Kubefirst download the templates, it will download the tag version that matches Kubefirst version
	if configs.K1Version != configs.DefaultK1Version {
		log.Info().Msg("loading tag values for built version")
		log.Info().Msgf("Kubefirst version %q, tags %q", configs.K1Version, config.K3dVersion)
		// in order to make the fallback tags work, set gitops branch as empty
		gitOpsBranch = ""
		templateTag = configs.K1Version
		viper.Set("template.tag", templateTag)
	}

	// set default values to kubefirst file
	viper.Set("gitops.repo", gitOpsRepo)
	viper.Set("gitops.owner", "kubefirst")
	viper.Set("gitprovider", pkg.GitHubProviderName)
	viper.Set("metaphor.branch", metaphorBranch)

	viper.Set("gitops.branch", gitOpsBranch)
	viper.Set("github.owner", viper.GetString("github.user"))
	viper.Set("cloud", pkg.CloudK3d)
	viper.Set("cluster-name", pkg.LocalClusterName)
	viper.Set("adminemail", adminEmail)

	viper.Set("argocd.local.service", pkg.ArgoCDLocalURL)
	viper.Set("vault.local.service", pkg.VaultLocalURL)

	// addons
	addon.AddAddon("github")
	addon.AddAddon("k3d")
	// used for letsencrypt notifications and the gitlab root account

	viper.Set("github.atlantis.webhook.secret", pkg.Random(20))

	err = viper.WriteConfig()
	if err != nil {
		return err
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

	progressPrinter.SetupProgress(6, silentMode)

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
	pkg.CreateSshKeyPair()
	log.Info().Msg("ssh key pair creation complete")
	progressPrinter.IncrementTracker("step-ssh", 1)

	repo.PrepareKubefirstTemplateRepo(
		dryRun,
		config,
		viper.GetString("github.owner"),
		viper.GetString("gitops.repo"),
		viper.GetString("gitops.branch"),
		viper.GetString("template.tag"),
	)
	log.Info().Msg("clone and detokenization of gitops-template repository complete")
	progressPrinter.IncrementTracker("step-gitops", 1)

	log.Info().Msg("sending init completed metric")

	pkg.InformUser("init is done!", silentMode)

	if useTelemetry {
		if err = wrappers.SendSegmentIoTelemetry("", pkg.MetricInitCompleted); err != nil {
			log.Error().Err(err).Msg("")
		}
	}

	progressPrinter.IncrementTracker("step-0", 1)
	time.Sleep(100 * time.Millisecond) // necessary to wait progress bar to finish

	return nil
}
