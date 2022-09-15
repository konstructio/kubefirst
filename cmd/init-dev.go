package cmd

import (
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/spf13/cobra"
	"log"
)

var intDevCmd = &cobra.Command{
	Use:   "init-dev",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {

		// todo:
		// - delete single record entry
		// - delete whole hosted zone
		// - destroy
		//    --destroy-hosted-zone
		//    --destroy-hosted-zone-keep-base-records
		hostedZone := ""
		//err := aws.DeleteRoute53EntriesTXT(hostedZone)
		//if err != nil {
		//	log.Println(err)
		//}
		err := aws.DeleteRoute53EntriesA(hostedZone)
		if err != nil {
			log.Println(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(intDevCmd)
}
