package cli

import (
	"github.com/kubefirst/kubefirst/cmd/cli/clean"
	"github.com/kubefirst/kubefirst/cmd/cli/cluster"
	"github.com/kubefirst/kubefirst/cmd/cli/initialization"
	"github.com/kubefirst/kubefirst/cmd/cli/local"
	"github.com/kubefirst/kubefirst/cmd/cli/tools"
	"github.com/kubefirst/kubefirst/cmd/cli/version"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	coreKubefirstCli := &cobra.Command{
		Use:   "kubefirst",
		Short: "Kubefirst CLI",
	}

	createCliCommandTree(coreKubefirstCli)

	return coreKubefirstCli
}

func createCliCommandTree(cmd *cobra.Command) {

	cmd.AddCommand(initialization.NewCommand())
	cmd.AddCommand(cluster.NewCommand())
	cmd.AddCommand(local.NewCommand())

	cmd.AddCommand(clean.NewCommand())
	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(tools.NewCommand())
}
