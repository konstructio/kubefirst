package cli

import (
	"github.com/kubefirst/kubefirst/cmd/cli/clean"
	"github.com/kubefirst/kubefirst/cmd/cli/cluster"
	"github.com/kubefirst/kubefirst/cmd/cli/initialization"
	"github.com/kubefirst/kubefirst/cmd/cli/tools"
	"github.com/kubefirst/kubefirst/cmd/cli/version"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	coreKubefirstCli := &cobra.Command{
		Use:   "kubefirst",
		Short: "rewiring proposal",
	}

	createCoreCliCommandTree(coreKubefirstCli)

	return coreKubefirstCli
}

func createCoreCliCommandTree(cmd *cobra.Command) {
	// todo: initialization should be initialization/initialization() conflict
	cmd.AddCommand(initialization.NewCommand())
	cmd.AddCommand(cluster.NewCommand())

	cmd.AddCommand(clean.NewCommand())
	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(tools.NewCommand())
}
