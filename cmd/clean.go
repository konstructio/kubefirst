package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "removes all kubefirst resources locally for new execution",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// todo delete the s3 buckets associated with the ~/.kubefirst file
		// todo ask for user input to verify deletion?
		config := configs.ReadConfig()

		log.Printf("removing %q and %q", config.KubeConfigPath, config.KubefirstConfigFilePath)
		// todo ask for user input to verify?
		err := os.RemoveAll(config.K1srtFolderPath)
		if err != nil {
			log.Panicf("unable to delete %q folder, error is: %s", config.K1srtFolderPath, err)
		}

		err = os.Remove(config.KubefirstConfigFilePath)
		if err != nil {
			log.Panicf("unable to delete %q file, error is: ", err)
		}
		log.Printf("removed %q and %q", config.KubeConfigPath, config.KubefirstConfigFilePath)

		log.Printf("%q and %q folders were removed", config.K1srtFolderPath, config.KubectlClientPath)

		if err := os.Mkdir(fmt.Sprintf("%s", config.K1srtFolderPath), os.ModePerm); err != nil {
			log.Panicf("error: could not create directory %q - it must exist to continue. error is: %s", config.K1srtFolderPath, err)
		}
		toolsDir := fmt.Sprintf("%s/tools", config.K1srtFolderPath)
		if err := os.Mkdir(toolsDir, os.ModePerm); err != nil {
			log.Panicf("error: could not create directory %q/tools - it must exist to continue %s", config.K1srtFolderPath, err)
		}

		log.Printf("created %q and %q/tools - proceed to `kubefirst init`", config.KubefirstConfigFilePath, config.K1srtFolderPath)
	},
}

func init() {
	initCmd.AddCommand(cleanCmd)
}
