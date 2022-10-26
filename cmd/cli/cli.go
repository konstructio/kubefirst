package cli

import (
	"github.com/kubefirst/kubefirst/cmd/cli/destroy"
	"github.com/kubefirst/kubefirst/cmd/cli/version"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	kubefirstCli := &cobra.Command{
		Use:   "kubefirst",
		Short: "rewiring proposal",
	}

	createCliCommandTree(kubefirstCli)

	return kubefirstCli
}

func createCliCommandTree(cmd *cobra.Command) {
	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(destroy.NewCommand())
	//cmd.AddCommand(clean.NewCommand())
	//cmd.AddCommand(info.NewCommand())
	//cmd.AddCommand(init.NewCommand())
	//cmd.AddCommand(create.NewCommand())
}
