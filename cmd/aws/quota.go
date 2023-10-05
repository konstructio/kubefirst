/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/kubefirst/kubefirst-api/pkg/reports"
	awsinternal "github.com/kubefirst/runtime/pkg/aws"
	"github.com/spf13/cobra"
)

// printAwsQuotaWarning provides visual output detailing quota health for aws
func printAwsQuotaWarning(messageHeader string, output map[string][]awsinternal.QuotaDetailResponse) string {
	var createAwsQuotaWarning bytes.Buffer
	createAwsQuotaWarning.WriteString(strings.Repeat("-", 70))
	createAwsQuotaWarning.WriteString(fmt.Sprintf("\n%s\n", messageHeader))
	createAwsQuotaWarning.WriteString(strings.Repeat("-", 70))
	createAwsQuotaWarning.WriteString("\n")
	for service, quotas := range output {
		createAwsQuotaWarning.WriteString(fmt.Sprintf("%s\n", service))
		createAwsQuotaWarning.WriteString(strings.Repeat("-", 35))
		createAwsQuotaWarning.WriteString("\n")
		for _, thing := range quotas {
			createAwsQuotaWarning.WriteString(fmt.Sprintf("%s: %v\n", thing.QuotaName, thing.QuotaValue))
		}
		createAwsQuotaWarning.WriteString("\n")
	}

	// Write to logs, but also output to stdout
	return createAwsQuotaWarning.String()

}

// evalAwsQuota provides an interface to the command-line
func evalAwsQuota(cmd *cobra.Command, args []string) error {
	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		return err
	}

	awsClient := &awsinternal.AWSConfiguration{
		Config: awsinternal.NewAwsV2(cloudRegionFlag),
	}
	quotaDetails, err := awsClient.GetServiceQuotas([]string{"eks", "vpc"})
	if err != nil {
		return err
	}

	var messageHeader = fmt.Sprintf(
		"AWS Quota Health\nRegion: %s\n\nIf you encounter issues deploying your kubefirst cluster, check these quotas and determine if you need to request a limit increase.",
		cloudRegionFlag,
	)
	result := printAwsQuotaWarning(messageHeader, quotaDetails)

	// Write to logs, but also output to stdout
	fmt.Println(reports.StyleMessage(result))

	return nil
}
