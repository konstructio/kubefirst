package cmd

import (
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a kubefirst management cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := createGitlabCmd.RunE(cmd, args)
		if err != nil {
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

}
