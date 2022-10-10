package cmd

import (
	"github.com/kubefirst/kubefirst/internal/addon"
	"github.com/spf13/cobra"
)

// addonListCmd represents the addonList command
var addonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubefirst Addons",
	Long:  `A list of Kubefirst addons, with this command you can see all the Addons suported by Kubefirst`,
	Run: func(cmd *cobra.Command, args []string) {
		addon.ListAddons()
	},
}

func init() {
	addonCmd.AddCommand(addonListCmd)
}
