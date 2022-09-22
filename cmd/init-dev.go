package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/spf13/cobra"
	"time"
)

var intDevCmd = &cobra.Command{
	Use:   "init-dev",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {

		// todo:
		// - delete single record entry [ok]
		// - delete whole hosted zone [ok]
		// - destroy
		//    --destroy-hosted-zone
		//    --destroy-hosted-zone-keep-base-records
		//
		// TXT: check if remove all feature ? yes-> delete all / no-> keep liveness [ok]
		// delete all hosted zone -> call delete TXT, records, [ok]
		//
		hostedZone := ""

		hostedZoneId, err := aws.Route53GetHostedZoneId(hostedZone)
		if err != nil {
			return err
		}

		txtRecords, err := aws.Route53ListTXTRecords(hostedZoneId)
		if err != nil {
			return err
		}

		fmt.Println("Number of records before the delete: ", len(txtRecords))

		keepLivenessRecord := false
		err = aws.Route53DeleteTXTRecords(hostedZoneId, hostedZone, keepLivenessRecord, txtRecords)
		if err != nil {
			return err
		}

		txtRecords, err = aws.Route53ListTXTRecords(hostedZoneId)
		if err != nil {
			return err
		}
		fmt.Println("Number of records after the delete: ", len(txtRecords))

		time.Sleep(3 * time.Second)

		//TXTRecord stores Route53 TXT record data
		aRecords, err := aws.Route53ListARecords(hostedZoneId)
		if err != nil {
			return err
		}

		fmt.Println("Number of records before the delete: ", len(aRecords))

		err = aws.Route53DeleteARecords(hostedZoneId, aRecords)
		if err != nil {
			return err
		}

		aRecords, err = aws.Route53ListARecords(hostedZoneId)
		if err != nil {
			return err
		}

		fmt.Println("Number of records after the delete: ", len(aRecords))

		fmt.Println("sleeping..")
		time.Sleep(3 * time.Second)

		err = aws.Route53DeleteHostedZone(hostedZoneId, hostedZone)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(intDevCmd)
}
