package cmd

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	awsinternal "github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String("us-east-1"),
		}))
		eksSvc := eks.New(sess)

		input := &eks.DescribeClusterInput{
			Name: aws.String("kubefirst-tech-4"),
		}
		eksClusterInfo, err := eksSvc.DescribeCluster(input)
		if err != nil {
			log.Info().Msgf("Error calling DescribeCluster: %v", err)
		}
		clientset, err := awsinternal.NewClientset(eksClusterInfo.Cluster)
		if err != nil {
			log.Info().Msgf("Error creating clientset: %v", err)
		}

		secData, err := k8s.ReadSecretV2(clientset, "argocd", "argocd-initial-admin-secret")
		fmt.Println(secData["password"])

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
