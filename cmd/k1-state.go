package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/state"
	"github.com/spf13/cobra"
	"log"
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

		bucketName, err := cmd.Flags().GetString("bucket-name")
		if err != nil {
			log.Panic(err)
		}

		if push {
			encryptFilename := "./testfile2.txt.bin"
			err := state.EncryptFile(encryptFilename)
			if err != nil {
				fmt.Println(err)
			}

			err = state.SendFileToS3(bucketName, encryptFilename)
			if err != nil {
				fmt.Println(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(k1state)

	k1state.Flags().Bool("push", false, "push Kubefirst config file to the S3 bucket")
	k1state.Flags().Bool("pull", false, "pull Kubefirst config file to the S3 bucket")
	k1state.Flags().String("bucket-name", "false", "set the bucket name to store the Kubefirst config file")

}
