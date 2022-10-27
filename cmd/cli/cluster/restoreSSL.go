package cluster

//
//import (
//	"fmt"
//	"github.com/kubefirst/kubefirst/cmd"
//	"log"
//
//	"github.com/kubefirst/kubefirst/internal/flagset"
//	"github.com/kubefirst/kubefirst/internal/ssl"
//	"github.com/spf13/cobra"
//)
//
//// restoreSSLCmd represents the restoreSSL command
//var restoreSSLCmd = &cobra.Command{
//	Use:   "restoreSSL",
//	Short: "Restore SSL certificates from a previous install",
//	Long:  `TBD`,
//	RunE: func(cmd *cobra.Command, args []string) error {
//		log.Println("Started restoreSSL")
//		fmt.Println("restoreSSL called")
//		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
//		if err != nil {
//			log.Println("Error restoreSSL global flags:", err)
//			return err
//		}
//
//		//includeMetaphorApps, err := cmd.Flags().GetBool("include-metaphor")
//		includeMetaphorApps := true
//		if err != nil {
//			log.Println("Error restoreSSL:", err)
//			return err
//		}
//		log.Println("RestoreSSL includeMetaphorApps:", includeMetaphorApps)
//
//		err = ssl.RestoreSSL(globalFlags.DryRun, includeMetaphorApps)
//		if err != nil {
//			fmt.Printf("Bucket not found, missing SSL backup, assuming first installation, error is: %v", err)
//			return err
//		}
//		return nil
//	},
//}
//
//func init() {
//	cmd.rootCmd.AddCommand(restoreSSLCmd)
//	//restoreSSLCmd.Flags().Bool("include-metaphor", false, "Include Metaphor Apps in process")
//	flagset.DefineGlobalFlags(restoreSSLCmd)
//}
