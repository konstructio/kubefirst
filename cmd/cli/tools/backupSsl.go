package tools

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/spf13/cobra"
)

// backupSslCmd represents the backupSsl command
var backupSslCmd = &cobra.Command{
	Use:   "backupSSL",
	Short: "Backup Secrets (cert-manager/certificates) to bucket kubefirst-<DOMAIN>",
	Long: `This command create a backupt of secrets from certmanager certificates to bucket named k1-<DOMAIN> 
where are using on provisioning phase with the flag`,

	RunE: func(cmd *cobra.Command, args []string) error {

		includeMetaphorApps, err := cmd.Flags().GetBool("include-metaphor")
		if err != nil {
			return err
		}

		_, err = ssl.GetBackupCertificates(includeMetaphorApps)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println("Backup certificates finished successfully")
		return nil
	},
}

func init() {
	backupSslCmd.Flags().Bool("include-metaphor", false, "Include Metaphor Apps in process")
}
