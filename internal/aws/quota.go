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
	sqTypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
)

type QuotaDetailResponse struct {
	QuotaName  string
	QuotaValue float64
}

// GetServiceQuotas
func (conf *AWSConfiguration) GetServiceQuotas(services []string) (map[string][]QuotaDetailResponse, error) {
	// Retrieve all quota information
	serviceQuotasClient := servicequotas.NewFromConfig(conf.Config)
	allQuotas := make(map[string][]sqTypes.ServiceQuota)
	returnQuotas := make(map[string][]QuotaDetailResponse)

	for _, service := range services {
		scopedQuotas := make([]sqTypes.ServiceQuota, 0)
		req := servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String(service),
		}

		for {
			resp, err := serviceQuotasClient.ListServiceQuotas(context.Background(), &req)
			if err != nil {
				return map[string][]QuotaDetailResponse{}, err
			}
			scopedQuotas = append(scopedQuotas, resp.Quotas...)
			req.NextToken = resp.NextToken
			if req.NextToken == nil {
				break
			}
		}
		allQuotas[service] = scopedQuotas
	}

	for service, quota := range allQuotas {
		mergedQuotas := make([]QuotaDetailResponse, 0)
		for _, value := range quota {
			mergedQuotas = append(mergedQuotas, QuotaDetailResponse{
				QuotaName:  *value.QuotaName,
				QuotaValue: *value.Value,
			})
		}
		returnQuotas[service] = mergedQuotas
	}

	return returnQuotas, nil

}
