package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/spf13/cobra"
)

// restoreSSLCmd represents the restoreSSL command
var restoreSSLCmd = &cobra.Command{
	Use:   "restoreSSL",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("restoreSSL called")
		err := ssl.RestoreSSL()
		if err != nil {
			fmt.Println("Bucket not found, missing SSL backup, assuming first installation")
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(restoreSSLCmd)
}
