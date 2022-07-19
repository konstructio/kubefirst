package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/spf13/cobra"
)

// backupSslCmd represents the backupSsl command
var backupSslCmd = &cobra.Command{
	Use:   "backupSSL",
	Short: "Backup Secrets (cert-manager/certificates) to bucket k1-<DOMAIN>",
	Long: `This command create a backupt of secrets from certmanager certificates to bucket named k1-<DOMAIN> 
where can be used on provisioning phase with the flag --recycle-ssl`,

	Run: func(cmd *cobra.Command, args []string) {
		_, err := ssl.GetBackupCertificates()
		if err != nil {
			log.Panic(err)
		}
		fmt.Println("Backup certificates finished successfully")
	},
}

func init() {
	rootCmd.AddCommand(backupSslCmd)
}
