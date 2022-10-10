package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addonCmd represents the addon command
var addonCmd = &cobra.Command{
	Use:   "addon",
	Short: "Addon Command - to manage Kubefirst supported addons",
	Long: `Commnad to manage kubefirst addons, list supported addons and more.

	To see all the supported addons type: kubefirst addons list `,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("addon root command - call kubefirst addon --help to see more options")
	},
}

func init() {
	rootCmd.AddCommand(addonCmd)
}
