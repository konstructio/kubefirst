/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

func (conf *AWSConfiguration) ListQuotas() (*servicequotas.GetServiceQuotaOutput, error) {
	quotasClient := servicequotas.NewFromConfig(conf.Config)

	quota, err := quotasClient.GetServiceQuota(context.Background(), &servicequotas.GetServiceQuotaInput{
		QuotaCode:   aws.String("L-DC2B2D3D"),
		ServiceCode: aws.String("s3"),
	})
	if err != nil {
		return &servicequotas.GetServiceQuotaOutput{}, err
	}

	return quota, nil
}
