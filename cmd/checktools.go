/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// checktoolsCmd represents the checktools command
var checktoolsCmd = &cobra.Command{
	Use:   "checktools",
	Short: "use to check compatibility of .kubefirst/tools",
	Long: `Execute a compatibility check of the tools downloaded by the installer.
	Execute After callint "init". If executed before init, tools will not be available. 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking the  tools installed used by installer:")

		kubectlVersion, kubectlStdErr,errKubectl := execShellReturnStrings(kubectlClientPath, "version", "--client", "--short")
		fmt.Printf("-> kubectl version:\n\t%s\n\t%s\n",kubectlVersion,kubectlStdErr)
		terraformVersion, terraformStdErr,errTerraform := execShellReturnStrings(terraformPath, "version")
		fmt.Printf("-> terraform version:\n\t%s\n\t%s\n",terraformVersion,terraformStdErr)
		helmVersion, helmStdErr,errHelm := execShellReturnStrings(helmClientPath, "version", "--client", "--short")
		fmt.Printf("-> helm version:\n\t%s\n\t%s\n",helmVersion,helmStdErr)

		if errKubectl != nil {
			fmt.Println("failed to call kubectlVersionCmd.Run(): %v", errKubectl)
		}
		if errHelm != nil {
			fmt.Println("failed to call helmVersionCmd.Run(): %v", errHelm)
		}
		if errTerraform != nil {
			fmt.Println("failed to call terraformVersionCmd.Run(): %v", errTerraform)
		}
		
	},
}

func init() {
	rootCmd.AddCommand(checktoolsCmd)
}
