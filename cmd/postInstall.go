package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/segmentio/analytics-go"
	"log"
	"runtime"
	"time"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/reports"

	"github.com/kubefirst/kubefirst/pkg"

	"github.com/spf13/cobra"
)

// preRunE is executed before the main command is called. It sends a new telemetry if it's allowed to.
func preRunE(cmd *cobra.Command, args []string) error {

	globalFlags, err := flagset.ProcessGlobalFlags(cmd)
	if err != nil {
		return err
	}

	awsFlags, err := flagset.ProcessAwsFlags(cmd)
	if err != nil {
		return err
	}

	createFlags, err := flagset.ProcessCreateFlags(cmd)
	if err != nil {
		return err
	}

	if createFlags.EnableConsole && globalFlags.UseTelemetry {

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
			pkg.MetricConsoleOpened,
			awsFlags.HostedZoneName,
			configs.K1Version,
		)
		if err != nil {
			log.Println(err)
		}
		telemetryService := services.NewSegmentIoService(segmentIOClient)
		telemetryHandler := handlers.NewTelemetryHandler(telemetryService)

		err = telemetryHandler.SendCountMetric(telemetryDomain)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

var postInstallCmd = &cobra.Command{
	Use:     "post-install",
	Short:   "starts post install process",
	PreRunE: preRunE,
	Long:    "Starts post install process to open the Console UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		createFlags, err := flagset.ProcessCreateFlags(cmd)
		if err != nil {
			return err
		}

		if createFlags.EnableConsole {
			log.Println("Starting the presentation of console and api for the handoff screen")
			go func() {
				errInThread := api.RunE(cmd, args)
				if errInThread != nil {
					log.Println(errInThread)
				}
			}()
			go func() {
				errInThread := console.RunE(cmd, args)
				if errInThread != nil {
					log.Println(errInThread)
				}
			}()

			log.Println("Kubefirst Console avilable at: http://localhost:9094", globalFlags.SilentMode)
		} else {
			log.Println("Skipping the presentation of console and api for the handoff screen")
		}

		openbrowser("http://localhost:9094")
		reports.HandoffScreen(globalFlags.DryRun, globalFlags.SilentMode)
		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(postInstallCmd)

	currentCommand := postInstallCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineCreateFlags(currentCommand)
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		_, _, err = pkg.ExecShellReturnStrings("xdg-open", url)
	case "windows":
		_, _, err = pkg.ExecShellReturnStrings("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		_, _, err = pkg.ExecShellReturnStrings("open", url)
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Println(err)
	}
}
