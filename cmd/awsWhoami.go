/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// awsWhoamiCmd represents the awsWhoami command
var awsWhoamiCmd = &cobra.Command{
	Use:   "aws-whoami",
	Short: "List the AWS ARN and Account ID",
	Long:  `This command lists the ARN and Account ID used by kubefirst at the time of execution`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("awsWhoami called")
		aws.GetAccountInfo()
		fmt.Println(viper.GetString("aws.accountid"))
		fmt.Println(viper.GetString("aws.userarn"))
	},
}

func init() {
	actionCmd.AddCommand(awsWhoamiCmd)
}
