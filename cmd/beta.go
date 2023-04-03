package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/cmd/vultr"
	"github.com/spf13/cobra"
)

// betaCmd represents the beta command tree
var betaCmd = &cobra.Command{
	Use:   "beta",
	Short: "access kubefirst beta features",
	Long:  `access kubefirst beta features`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("To learn more about kubefirst, run:")
		fmt.Println("  kubefirst help")
	},
}

func init() {
	cobra.OnInitialize()
	betaCmd.AddCommand(
		vultr.NewCommand(),
	)
}
