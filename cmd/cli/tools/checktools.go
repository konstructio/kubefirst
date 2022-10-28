package tools

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
)

// checktoolsCmd represents the checktools command
var checktoolsCmd = &cobra.Command{
	Use:   "checktools",
	Short: "use to check compatibility of .kubefirst/tools",
	Long: `Execute a compatibility check of the tools downloaded by the installer.
	Execute After call "init". If executed before init, tools will not be available. 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		config := configs.ReadConfig()

		fmt.Println("Checking the  tools installed used by installer:")

		kubectlVersion, kubectlStdErr, errKubectl := pkg.ExecShellReturnStrings(config.KubectlClientPath, "version", "--client", "--short")
		fmt.Printf("-> kubectl version:\n\t%s\n\t%s\n", kubectlVersion, kubectlStdErr)
		terraformVersion, terraformStdErr, errTerraform := pkg.ExecShellReturnStrings(config.TerraformClientPath, "version")
		fmt.Printf("-> terraform version:\n\t%s\n\t%s\n", terraformVersion, terraformStdErr)
		helmVersion, helmStdErr, errHelm := pkg.ExecShellReturnStrings(config.HelmClientPath, "version", "--client", "--short")
		fmt.Printf("-> helm version:\n\t%s\n\t%s\n", helmVersion, helmStdErr)

		if errKubectl != nil {
			fmt.Printf("failed to call kubectlVersionCmd.Run(): %v", errKubectl)
		}
		if errHelm != nil {
			fmt.Printf("failed to call helmVersionCmd.Run(): %v", errHelm)
		}
		if errTerraform != nil {
			fmt.Printf("failed to call terraformVersionCmd.Run(): %v", errTerraform)
		}

	},
}

func init() {
	//cmd.rootCmd.AddCommand(checktoolsCmd)
}
