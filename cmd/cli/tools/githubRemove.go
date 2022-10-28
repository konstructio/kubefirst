package tools

import (
	"github.com/kubefirst/kubefirst/pkg"
	"log"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// githubRemoveCmd represents the githubRemove command
var githubRemoveCmd = &cobra.Command{
	Use:   "remove-github",
	Short: "Undo github setup",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, githubRemoveCmd skipped.")
			return nil
		}

		log.Println("Org used:", viper.GetString("github.org"))
		log.Println("dry-run:", globalFlags.DryRun)

		if viper.GetBool("github.terraformapplied.gitops") {

			pkg.InformUser("Destroying repositories with terraform in GitHub", globalFlags.SilentMode)

			github.DestroyGitHubTerraform(globalFlags.DryRun)

			pkg.InformUser("GitHub terraform destroyed", globalFlags.SilentMode)
		}

		log.Printf("GitHub terraform Executed with Success")
		return nil
	},
}

func init() {
	currentCommand := githubRemoveCmd
	flagset.DefineGlobalFlags(currentCommand)
}
