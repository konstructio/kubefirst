package local

import (
	"github.com/kubefirst/kubefirst/cmd"
	"github.com/kubefirst/kubefirst/cmd/cli/cluster"
	"github.com/kubefirst/kubefirst/cmd/cli/initialization"
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

		initFlags := initialization.InitCmd.Flags()
		//err := initFlags.Set("gitops-branch", "main")
		err := initFlags.Set("gitops-branch", "main")
		if err != nil {
			return err
		}
		//viper.Set("gitops.branch", "main")
		viper.Set("gitops.branch", "main")

		err = initFlags.Set("metaphor-branch", "main")
		if err != nil {
			return err
		}
		viper.Set("metaphor.branch", "main")

		err = viper.WriteConfig()
		if err != nil {
			return err
		}

		err = initialization.InitCmd.ParseFlags(args)
		if err != nil {
			return err
		}

		err = initialization.InitCmd.RunE(cmd, args)
		if err != nil {
			return err
		}

		// create
		if err = cluster.createCmd.Flags().Set("enable-console", "true"); err != nil {
			return err
		}

		viper.Set("metaphor.branch", "main")
		viper.Set("botpassword", "kubefirst-123")
		viper.Set("adminemail", "joao@kubeshop.io")
		err = cluster.createCmd.RunE(cmd, args)
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

	cmd.rootCmd.AddCommand(localCmd)
	currentCommand := localCmd
	//log.SetPrefix("LOG: ")
	//log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineGithubCmdFlags(currentCommand)
	flagset.DefineAWSFlags(currentCommand)
	flagset.DefineInstallerGenericFlags(currentCommand)

}