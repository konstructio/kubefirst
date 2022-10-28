package tools

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "",
		Long:  "",
	}

	toolsCmd.AddCommand(InfoCommand())
	toolsCmd.AddCommand(awsWhoamiCommand())

	return toolsCmd
}
