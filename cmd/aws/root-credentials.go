/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/kubefirst/runtime/pkg/credentials"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getAwsRootCredentials(cmd *cobra.Command, args []string) error {
	cloudRegion := viper.GetString("flags.cloud-region")
	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	// Parse flags
	a, err := cmd.Flags().GetBool("argocd")
	if err != nil {
		return err
	}
	k, err := cmd.Flags().GetBool("kbot")
	if err != nil {
		return err
	}
	v, err := cmd.Flags().GetBool("vault")
	if err != nil {
		return err
	}
	opts := credentials.CredentialOptions{
		CopyArgoCDPasswordToClipboard: a,
		CopyKbotPasswordToClipboard:   k,
		CopyVaultPasswordToClipboard:  v,
	}

	// Determine if there are active installs
	_, err = credentials.EvalAuth(awsinternal.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Instantiate kubernetes client
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cloudRegion),
	}))

	eksSvc := eks.New(sess)

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}
	eksClusterInfo, err := eksSvc.DescribeCluster(clusterInput)
	if err != nil {
		log.Fatal().Msgf("error calling DescribeCluster: %v", err)
	}

	clientset, err := awsinternal.NewClientset(eksClusterInfo.Cluster)
	if err != nil {
		log.Fatal().Msgf("error creating clientset: %v", err)
	}

	err = credentials.ParseAuthData(clientset, awsinternal.CloudProvider, gitProvider, domainName, &opts)
	if err != nil {
		return err
	}

	return nil
}
