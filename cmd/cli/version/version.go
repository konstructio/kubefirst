package version

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "print the version number for kubefirst-cli",
		Long:  `All software has versions. This is kubefirst's`,
		Run:   runVersionCmd,
	}
	return versionCmd
}

func runVersionCmd(cmd *cobra.Command, args []string) {
	fmt.Printf("\n\nkubefirst-cli golang utility version: %s\n\n", configs.K1Version)
}
