package local

import (
	"github.com/kubefirst/kubefirst/cmd/cli/cluster"
	"github.com/kubefirst/kubefirst/cmd/cli/initialization"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	gitOpsBranch   string
	metaphorBranch string
	gitProvider    string
	cloud          string
	enableConsole  bool
	dryRun         bool
)

func NewCommand() *cobra.Command {
	localCmd := &cobra.Command{
		Use:   "local",
		Short: "Kubefirst localhost installation",
		Long:  "Kubefirst localhost enable a localhost installation without the requirement of a cloud provider.",
		RunE:  runLocalCommand,
	}

	localCmd.Flags().StringVar(&gitOpsBranch, "gitops-branch", "main", "")
	localCmd.Flags().StringVar(&metaphorBranch, "metaphor-branch", "main", "")
	localCmd.Flags().StringVar(&gitProvider, "git-provider", "github", "")
	localCmd.Flags().StringVar(&cloud, "cloud", "k3d", "")
	localCmd.Flags().BoolVar(&enableConsole, "enable-console", true, "")
	localCmd.Flags().BoolVar(&dryRun, "dry-run", false, "")

	return localCmd
}

func runLocalCommand(cmd *cobra.Command, args []string) error {

	viper.Set("gitops.branch", gitOpsBranch)
	viper.Set("metaphor.branch", metaphorBranch)

	err := viper.WriteConfig()
	if err != nil {
		return err
	}

	err = initialization.RunInit(cmd, args)
	if err != nil {
		return err
	}

	// create
	if err = cluster.CreateCommand().Flags().Set("enable-console", "true"); err != nil {
		return err
	}

	viper.Set("metaphor.branch", "main")
	viper.Set("botpassword", "kubefirst-123")
	viper.Set("adminemail", "joao@kubeshop.io")
	err = cluster.CreateCommand().RunE(cmd, args)
	if err != nil {
		return err
	}

	return nil
}
