package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// backupSslCmd represents the backupSsl command
var backupSslCmd = &cobra.Command{
	Use:   "backupSSL",
	Short: "Backup Secrets (cert-manager/certificates) to bucket kubefirst-<DOMAIN>",
	Long: `This command create a backupt of secrets from certmanager certificates to bucket named kubefirst-<DOMAIN> 
where can be used on provisioning phase with the flag --recycle-ssl`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("backupSsl called")
		bucketName := fmt.Sprintf("k1-%s", viper.GetString("aws.hostedzonename"))
		aws.CreateBucket(false, bucketName)

		namespaces := []string{"argo", "atlantis", "chartmuseum", "gitlab", "vault"}
		files, err := ssl.GetBackupCertificates(namespaces)
		if err != nil {
			log.Panic(err)
		}

		for _, v := range files {
			fullPath := fmt.Sprintf("%s/%s", "cert-manager", v)
			aws.UploadFile(bucketName, fullPath, v)
		}
	},
}

func init() {
	rootCmd.AddCommand(backupSslCmd)
}
