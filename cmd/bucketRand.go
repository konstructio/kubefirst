/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cip8/autoname"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// bucketRandCmd represents the bucketRand command
var bucketRandCmd = &cobra.Command{
	Use:   "bucketRand",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		// cfg, err := config.LoadDefaultConfig(context.TODO())
		// if err != nil {
		// 	fmt.Println("failed to load configuration, error:", err)
		// }
		// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(viper.GetString("aws.region"))},
		)

		s3Client := s3.New(sess)

		randomName := strings.ReplaceAll(autoname.Generate(), "_", "-")
		viper.Set("bucket.rand", randomName)

		buckets := strings.Fields("state-store argo-artifacts gitlab-backup chartmuseum")
		for _, bucket := range buckets {
			bucketExists := viper.GetBool(fmt.Sprintf("bucket.%s.created", bucket))
			if bucketExists != true {
				bucketName := fmt.Sprintf("k1-%s-%s", bucket, randomName)

				fmt.Println("creating", bucket, "bucket", bucketName)

				regionName := viper.GetString("aws.region")
				fmt.Println("region is ", regionName)
				if regionName == "us-east-1" {
					_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
						Bucket: &bucketName,
					})
				} else {
					_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
						Bucket: &bucketName,
						CreateBucketConfiguration: &s3.CreateBucketConfiguration{
							LocationConstraint: aws.String(regionName),
						},
					})
				}

				if err != nil {
					fmt.Println("failed to create bucket "+bucketName, err.Error())
					os.Exit(1)
				}
				viper.Set(fmt.Sprintf("bucket.%s.created", bucket), true)
				viper.Set(fmt.Sprintf("bucket.%s.name", bucket), bucketName)
				viper.WriteConfig()
			}
			fmt.Println(fmt.Sprintf("bucket %s exists", viper.GetString(fmt.Sprintf("bucket.%s.name", bucket))))
		}
	},
}

func init() {
	nebulousCmd.AddCommand(bucketRandCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bucketRandCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bucketRandCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
