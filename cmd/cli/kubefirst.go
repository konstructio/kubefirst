package cli

import (
	"github.com/kubefirst/kubefirst/cmd/cli/cluster"
	"github.com/kubefirst/kubefirst/cmd/cli/prepare"
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
	// todo: prepare should be init/init() conflict
	cmd.AddCommand(prepare.NewCommand())
	cmd.AddCommand(cluster.NewCommand())

	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(tools.NewCommand())
}
