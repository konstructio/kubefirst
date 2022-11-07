package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/cmd/local"
	"github.com/kubefirst/kubefirst/configs"
	"os"

	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubefirst",
	Short: "kubefirst management cluster installer base command",
	Long: `kubefirst management cluster installer provisions an
	open source application delivery platform in under an hour. 
	checkout the docs at docs.kubefirst.io.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// wire viper config for flags for all commands
		return configs.InitializeViperConfig(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("To learn more about kubefirst, run:")
		fmt.Println("  kubefirst help")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	//This will allow all child commands to have informUser available for free.
	//Refers: https://github.com/kubefirst/kubefirst/issues/525
	//Before removing next line, please read ticket above.
	progressPrinter.GetInstance()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()
	rootCmd.AddCommand(local.NewCommand())
}
