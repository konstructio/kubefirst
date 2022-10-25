package cmd

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/reports"

	"github.com/kubefirst/kubefirst/pkg"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var postInstallCmd = &cobra.Command{
	Use:   "post-install",
	Short: "starts post install process",
	Long:  "Starts post install process to open the Console UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("---debug---")
		fmt.Println("this was called (this later)")
		fmt.Println("---debug---")
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

		if viper.GetString("cloud") == flagset.CloudK3d {
			reports.LocalHandoffScreen(globalFlags.DryRun, globalFlags.SilentMode)
		} else {
			reports.HandoffScreen(globalFlags.DryRun, globalFlags.SilentMode)
		}

		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(postInstallCmd)

	flagset.DefineGlobalFlags(postInstallCmd)
	postInstallCmd.Flags().Bool("enable-console", true, "If hand-off screen will be presented on a browser UI")
	//flagset.DefineCreateFlags(currentCommand)
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
