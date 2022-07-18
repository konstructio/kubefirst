package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var k1state = &cobra.Command{
	Use:   "state",
	Short: "push and pull Kubefirst configuration to S3 bucket",
	Long:  `Kubefirst configuration can be handed over to another user by pushing the Kubefirst config files to a S3 bucket.`,
	Run: func(cmd *cobra.Command, args []string) {

		push, err := cmd.Flags().GetBool("push")
		if err != nil {
			log.Panic(err)
		}
		pull, err := cmd.Flags().GetBool("pull")
		if err != nil {
			log.Panic(err)
		}

		bucketName, err := cmd.Flags().GetString("bucket-name")
		if err != nil {
			log.Panic(err)
		}

		if !push && !pull {
			fmt.Println(cmd.Help())
			return
		}

		config := configs.ReadConfig()
		if push {
			//encryptFilename := "./tmp/"
			//err := state.EncryptFile(encryptFilename)
			//if err != nil {
			//	fmt.Println(err)
			//	return
			//}

			err = aws.SendFileToS3(bucketName, config.KubefirstConfigFilePath, config.KubefirstConfigFileName)
			if err != nil {
				fmt.Println(err)
				return
			}
			finalMsg := fmt.Sprintf("Kubefirst configuration file was upload to AWS S3 at %q bucket name", bucketName)

			log.Printf(finalMsg)
			fmt.Println(reports.StyleMessage(finalMsg))
		}

		if pull {
			err := aws.DownloadS3File(bucketName, config.KubefirstConfigFileName)
			if err != nil {
				fmt.Println(err)
				return
			}
			currentFolder, err := os.Getwd()
			finalMsg := fmt.Sprintf("Kubefirst configuration file was downloaded to %q/, and is now available to be copied to %q/",
				currentFolder,
				config.K1FolderPath,
			)

			log.Printf(finalMsg)
			fmt.Println(reports.StyleMessage(finalMsg))
		}
	},
}

func init() {
	rootCmd.AddCommand(k1state)

	k1state.Flags().Bool("push", false, "push Kubefirst config file to the S3 bucket")
	k1state.Flags().Bool("pull", false, "pull Kubefirst config file to the S3 bucket")
	k1state.Flags().String("bucket-name", "", "set the bucket name to store the Kubefirst config file")
	err := k1state.MarkFlagRequired("bucket-name")
	if err != nil {
		log.Println(err)
		return
	}

}
