package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// domainName := viper.GetString("domain-name")
		k1Dir := viper.GetString("kubefirst.k1-dir")

		fmt.Printf("checking path %s for ssl certificates\n", k1Dir+"/ssl/kubefast.com")
		if _, err := os.Stat(k1Dir + "/ssl/kubefast.com"); os.IsNotExist(err) {
			// path/to/whatever does not exist
			fmt.Println("path did NOT exist")
		} else {
			fmt.Println("path did exist")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
