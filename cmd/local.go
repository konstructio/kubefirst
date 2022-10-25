package cmd

import (
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// localCmd represents the init command
var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Kubefirst localhost installation",
	Long:  "Kubefirst localhost enable a localhost installation without the requirement of a cloud provider.",
	RunE: func(cmd *cobra.Command, args []string) error {

		initFlags := initCmd.Flags()
		//err := initFlags.Set("gitops-branch", "main")
		err := initFlags.Set("gitops-branch", "update_atlantis_chart_version")
		if err != nil {
			return err
		}
		//viper.Set("gitops.branch", "main")
		viper.Set("gitops.branch", "update_atlantis_chart_version")

		err = initFlags.Set("metaphor-branch", "main")
		if err != nil {
			return err
		}
		viper.Set("metaphor.branch", "main")

		err = viper.WriteConfig()
		if err != nil {
			return err
		}

		err = initCmd.ParseFlags(args)
		if err != nil {
			return err
		}

		err = initCmd.RunE(cmd, args)
		if err != nil {
			return err
		}

		// create
		if err = createCmd.Flags().Set("enable-console", "true"); err != nil {
			return err
		}

		viper.Set("metaphor.branch", "main")
		viper.Set("botpassword", "kubefirst-123")
		viper.Set("adminemail", "joao@kubeshop.io")
		err = createCmd.RunE(cmd, args)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {

	// Do we need this?
	//localCmd.Flags().Bool("clean", false, "delete any local kubefirst content ~/.kubefirst, ~/.k1")

	//Group Flags

	rootCmd.AddCommand(localCmd)
	currentCommand := localCmd
	//log.SetPrefix("LOG: ")
	//log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineGithubCmdFlags(currentCommand)
	flagset.DefineAWSFlags(currentCommand)
	flagset.DefineInstallerGenericFlags(currentCommand)

}
