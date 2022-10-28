package cluster

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Cluster level operations",
		Long:  "Provides cluster operations like create and destroy",
	}

	clusterCmd.AddCommand(CreateCommand())
	clusterCmd.AddCommand(CreateGitHubCommand())
	clusterCmd.AddCommand(DestroyCommand())

	return clusterCmd
}
