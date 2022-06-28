/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// callCmd represents the call command
var callCmd = &cobra.Command{
	Use:   "call",
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
		stsClient := sts.NewFromConfig(cfg)
		iamCaller, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
		if err != nil {
			fmt.Println("oh no error on call", err)
		}

		viper.Set("aws.accountid", *iamCaller.Account)
		viper.Set("aws.userarn", *iamCaller.Arn)
		viper.WriteConfig()
	},
}

func init() {
	nebulousCmd.AddCommand(callCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// callCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// callCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
