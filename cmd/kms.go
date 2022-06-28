/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/spf13/cobra"
)

// kmsCmd represents the kms command
var kmsCmd = &cobra.Command{
	Use:   "kms",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			fmt.Println("failed to load configuration, error:", err)
		}
		// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
		kmsClient := kms.NewFromConfig(cfg)
		kmsList, err := kmsClient.ListKeys(context.TODO(), &kms.ListKeysInput{})
		for _, k := range kmsList.Keys {
			fmt.Println(*k.KeyId)

		}

	},
}

func init() {
	nebulousCmd.AddCommand(kmsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kmsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kmsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
