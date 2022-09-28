package cmd

import (
	"log"

	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/state"

	"github.com/kubefirst/kubefirst/internal/flagset"
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

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
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
		// Relates to issue: https://github.com/kubefirst/kubefirst/issues/386
		// Metaphor needs chart museum for CI works
		informUser("Waiting chartmuseum", globalFlags.SilentMode)
		for i := 1; i < 10; i++ {
			chartMuseum := gitlab.AwaitHostNTimes("chartmuseum", globalFlags.DryRun, 20)
			if chartMuseum {
				informUser("Chartmuseum DNS is ready", globalFlags.SilentMode)
				break
			}
		}

		informUser("Removing self-signed Argo certificate", globalFlags.SilentMode)
		err = argocd.RemoveSelfSignedCert()
		if err != nil {
			log.Printf("Error removing self-signed certificate from ArgoCD: %s", err)
			return err
		}

		informUser("Checking if cluster is ready for use by metaphor apps", globalFlags.SilentMode)
		for i := 1; i < 10; i++ {
			err = k1ReadyCmd.RunE(cmd, args)
			if err != nil {
				log.Println(err)
			} else {
				break
			}
		}
		informUser("Deploying metaphor applications", globalFlags.SilentMode)
		err = deployMetaphorCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error deploy metaphor applications", globalFlags.SilentMode)
			log.Println("Error running deployMetaphorCmd")
			return err
		}
		err = state.UploadKubefirstToStateStore(globalFlags.DryRun)
		if err != nil {
			log.Println(err)
		}

		sendCompleteInstallTelemetry(globalFlags.DryRun, globalFlags.UseTelemetry)
		log.Println("Kubefirst installation finished successfully")
		informUser("Kubefirst installation finished successfully", globalFlags.SilentMode)

		err = postInstallCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error starting apps from post-install", globalFlags.SilentMode)
			log.Println("Error running postInstallCmd")
			return err
		}

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
	flagset.DefineCreateFlags(currentCommand)

}
