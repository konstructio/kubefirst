package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubefirst",
	Short: "kubefirst management cluster installer base command",
	Long: `kubefirst management cluster installer provisions an
	open source application delivery platform in under an hour. 
	checkout the docs at docs.kubefirst.com.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println(viper.Get("name"))
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
	rootCmd.Flags().Bool("disable-telemetry", false, "Installer will not send telemetry about this installation")
}
