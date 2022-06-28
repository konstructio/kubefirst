/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
)

// vaultCmd represents the vault command
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("vault called")

		config := vault.DefaultConfig()

		config.Address = "https://vault.kubefirst.io"

		client, err := vault.NewClient(config)
		if err != nil {
			fmt.Println("unable to initialize Vault client: %v", err)
		}

		// Authentication
		// client.SetToken("dev-only-token")

		secretData := map[string]interface{}{
			"data": map[string]interface{}{
				"password": "Hashi123",
			},
		}

		// Writing a secret
		_, err = client.Logical().Write("secret/data/my-secret-password", secretData)
		if err != nil {
			fmt.Println("unable to write secret: %v", err)
		}

		fmt.Println("Secret written successfully.")

	},
}

func init() {
	nebulousCmd.AddCommand(vaultCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// vaultCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// vaultCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
