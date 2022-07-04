package flare

import (
	"context"

	"log"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go/aws"
)

func DescribeCluster() {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("failed to load configuration, error:", err)
	}
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	eksClient := eks.NewFromConfig(cfg, func(o *eks.Options) {
		o.Region = "us-east-2"
	})

	cluster, err := eksClient.DescribeCluster(context.TODO(), &eks.DescribeClusterInput{
		Name: aws.String("kubefirst"),
	})
	if err != nil {
		log.Println("error describing cluster", err)
	}
	// todo base64 encoded data : *cluster.Cluster.CertificateAuthority.Data,
	log.Println("cluster:", *cluster.Cluster.Arn, *cluster.Cluster.Endpoint)
}
