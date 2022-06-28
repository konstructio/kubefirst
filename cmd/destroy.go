/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("destroy called")
		// todo this needs to be removed when we are no longer in the starter account
		os.Setenv("AWS_PROFILE", "starter")

		os.Setenv("AWS_REGION", viper.GetString("aws.region"))
		os.Setenv("AWS_ACCOUNT_ID", viper.GetString("aws.accountid"))
		os.Setenv("HOSTED_ZONE_NAME", viper.GetString("aws.domainname"))
		os.Setenv("GITLAB_TOKEN", viper.GetString("gitlab.token"))

		//! terraform destroy gitlab
		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)
		err := os.Chdir(directory)
		if err != nil {
			fmt.Println("error changing dir: ", directory)
		}

		os.Setenv("GITLAB_BASE_URL", fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname")))

		tfInitGitlabCmd := exec.Command(terraformPath, "init")
		tfInitGitlabCmd.Stdout = os.Stdout
		tfInitGitlabCmd.Stderr = os.Stderr
		err = tfInitGitlabCmd.Run()
		if err != nil {
			fmt.Println("failed to call terraform init gitlab: ", err)
			panic("failed to terraform init gitlab")
		}

		tfDestroyGitlabCmd := exec.Command(terraformPath, "destroy", "-auto-approve")
		tfDestroyGitlabCmd.Stdout = os.Stdout
		tfDestroyGitlabCmd.Stderr = os.Stderr
		err = tfDestroyGitlabCmd.Run()
		if err != nil {
			fmt.Println("failed to call terraform destroy gitlab: ", err)
			panic("failed to terraform destroy gitlab")
		}

		viper.Set("destroy.terraformdestroy.gitlab", true)
		viper.WriteConfig()

		//! terraform destroy base
		directory = fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		err = os.Chdir(directory)
		if err != nil {
			fmt.Println("error changing dir: ", directory)
		}

		tfInitBaseCmd := exec.Command(terraformPath, "init")
		tfInitBaseCmd.Stdout = os.Stdout
		tfInitBaseCmd.Stderr = os.Stderr
		err = tfInitBaseCmd.Run()
		if err != nil {
			fmt.Println("failed to call terraform init base: ", err)
		}

		tfDestroyBaseCmd := exec.Command(terraformPath, "destroy", "-auto-approve")
		tfDestroyBaseCmd.Stdout = os.Stdout
		tfDestroyBaseCmd.Stderr = os.Stderr
		err = tfDestroyBaseCmd.Run()
		if err != nil {
			fmt.Println("failed to call terraform destroy base: ", err)
			panic("failed to terraform destroy base")
		}

		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
	},
}

func init() {
	nebulousCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
