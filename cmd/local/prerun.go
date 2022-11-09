package local

import (
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/addon"
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
)

func validateLocal(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()

	log.Println("sending init started metric")

	var telemetryHandler handlers.TelemetryHandler
	if useTelemetry {
		// Instantiates a SegmentIO client to use send messages to the segment API.
		segmentIOClient := analytics.New(pkg.SegmentIOWriteKey)

		// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
		// to force flush in case there is still some pending message in the SegmentIO library queue.
		defer func(segmentIOClient analytics.Client) {
			err := segmentIOClient.Close()
			if err != nil {
				log.Println(err)
			}
		}(segmentIOClient)

		// validate telemetryDomain data
		telemetryDomain, err := domain.NewTelemetry(
			pkg.MetricInitStarted,
			awsHostedZone,
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		telemetryService := services.NewSegmentIoService(segmentIOClient)
		telemetryHandler = handlers.NewTelemetryHandler(telemetryService)

		err = telemetryHandler.SendCountMetric(telemetryDomain)
		if err != nil {
			log.Println(err)
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

	// todo: set constants
	viper.Set("argocd.local.service", "http://localhost:8080")
	viper.Set("gitlab.local.service", "http://localhost:8888")
	viper.Set("vault.local.service", "http://localhost:8200")
	addon.AddAddon("github")
	addon.AddAddon("k3d")
	// used for letsencrypt notifications and the gitlab root account

	atlantisWebhookSecret := pkg.Random(20)
	viper.Set("github.atlantis.webhook.secret", atlantisWebhookSecret)

	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	// todo: wrap business logic into the handler
	httpClient := http.DefaultClient
	gitHubService := services.NewGitHubService(httpClient)
	gitHubHandler := handlers.NewGitHubHandler(gitHubService)
	if config.GitHubPersonalAccessToken == "" {
		gitHubAccessToken, err := gitHubHandler.AuthenticateUser()
		if err != nil {
			return err
		}

		if gitHubAccessToken == "" {
			return errors.New("unable to retrieve a GitHub token for the user")
		}

		// todo: set common way to load env. values (viper->struct->load-env)
		// todo: use viper file to load it, not load env. value
		if err := os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
			return err
		}
		log.Println("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
	}

	// get GitHub data to set user and owner based on the provided token
	githubUser := gitHubHandler.GetGitHubUser(config.GitHubPersonalAccessToken)

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

	progressPrinter.SetupProgress(8, silentMode)

	progressPrinter.AddTracker("step-0", "Process Parameters", 1)
	progressPrinter.AddTracker("step-download", pkg.DownloadDependencies, 3)
	progressPrinter.AddTracker("step-gitops", pkg.CloneAndDetokenizeGitOpsTemplate, 1)
	progressPrinter.AddTracker("step-ssh", pkg.CreateSSHKey, 1)

	log.Println("installing kubefirst dependencies")
	progressPrinter.IncrementTracker("step-download", 1)
	err = downloadManager.DownloadTools(config)
	if err != nil {
		return err
	}
	log.Println("dependency installation complete")
	progressPrinter.IncrementTracker("step-download", 1)
	err = downloadManager.DownloadLocalTools(config)
	if err != nil {
		return err
	}

	progressPrinter.IncrementTracker("step-download", 1)

	log.Println("creating an ssh key pair for your new cloud infrastructure")
	pkg.CreateSshKeyPair()
	log.Println("ssh key pair creation complete")
	progressPrinter.IncrementTracker("step-ssh", 1)

	repo.PrepareKubefirstTemplateRepo(
		dryRun,
		config,
		viper.GetString("github.owner"),
		viper.GetString("gitops.repo"),
		viper.GetString("gitops.branch"),
		viper.GetString("template.tag"),
	)
	log.Println("clone and detokenization of gitops-template repository complete")
	progressPrinter.IncrementTracker("step-gitops", 1)

	log.Println("sending init completed metric")

	pkg.InformUser("init is done!\n", silentMode)

	if useTelemetry {
		telemetryInitCompleted, err := domain.NewTelemetry(
			pkg.MetricInitCompleted,
			awsHostedZone,
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		err = telemetryHandler.SendCountMetric(telemetryInitCompleted)
		if err != nil {
			log.Println(err)
		}
	}

	progressPrinter.IncrementTracker("step-0", 1)

	return nil
}
