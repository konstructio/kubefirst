/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// httpHealthCheckCmd represents the httpHealthCheck command
var httpHealthCheckCmd = &cobra.Command{
	Use:   "httpHealthCheck",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("httpHealthCheck called")

		// todo should this be aws.hostedzonedname since we're sticking to an
		// todo aws: and gcp: figure their nomenclature is more familar
		hostedZoneName := viper.GetString("aws.domainname")

		resp, err := http.Get(fmt.Sprintf("https://gitlab.%s", hostedZoneName))
		if err != nil {
			fmt.Println("error GET to gitlab")
		}

		fmt.Println(resp.StatusCode) //! need to ensure a 200 before continuing for loop

	},
}

func init() {
	nebulousCmd.AddCommand(httpHealthCheckCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// httpHealthCheckCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// httpHealthCheckCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
