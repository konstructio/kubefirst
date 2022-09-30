package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubefirst",
	Short: "kubefirst management cluster installer base command",
	Long: `kubefirst management cluster installer provisions an
	open source application delivery platform in under an hour. 
	checkout the docs at docs.kubefirst.io.`,
	Run: func(cmd *cobra.Command, args []string) {
		//log.Println(viper.Get("name"))
		fmt.Println("To learn more about kubefirst, run:")
		fmt.Println("  kubefirst help")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()

	// Cobra also supports local flags, which will only run, when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
