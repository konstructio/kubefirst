package cmd

import (
	"github.com/kubefirst/kubefirst/internal/state"
	"log"
	"time"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a kubefirst management cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}
		sendStartedInstallTelemetry(globalFlags.DryRun, globalFlags.UseTelemetry)
		if viper.GetBool("github.enabled") {
			log.Println("Installing Github version of Kubefirst")
			viper.Set("git.mode", "github")
			err := createGithubCmd.RunE(cmd, args)
			if err != nil {
				return err
			}

		} else {
			log.Println("Installing GitLab version of Kubefirst")
			viper.Set("git.mode", "gitlab")
			err := createGitlabCmd.RunE(cmd, args)
			if err != nil {
				return err
			}

		}
		informUser("Deploying metaphor applications")
		err = deployMetaphorCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error deploy metaphor applications")
			log.Println("Error running deployMetaphorCmd")
			return err
		}
		err = state.UploadKubefirstToStateStore(globalFlags.DryRun)
		if err != nil {
			log.Println(err)
		}

		sendCompleteInstallTelemetry(globalFlags.DryRun, globalFlags.UseTelemetry)
		reports.HandoffScreen(globalFlags.DryRun)
		time.Sleep(time.Millisecond * 2000)
		log.Println("End of creation run")
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createCmd)
	currentCommand := createCmd
	// todo: make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	createCmd.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
	flagset.DefineGlobalFlags(currentCommand)

}
