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

	awsinternal "github.com/konstructio/kubefirst-api/pkg/aws"
	"github.com/konstructio/kubefirst-api/pkg/reports"
	"github.com/spf13/cobra"
)

// printAwsQuotaWarning provides visual output detailing quota health for aws
func printAwsQuotaWarning(messageHeader string, output map[string][]awsinternal.QuotaDetailResponse) string {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, strings.Repeat("-", 70))
	fmt.Fprintln(&buf, messageHeader)
	fmt.Fprintln(&buf, strings.Repeat("-", 70))
	fmt.Fprintln(&buf, "")

	for service, quotas := range output {
		fmt.Fprintln(&buf, service)
		fmt.Fprintln(&buf, strings.Repeat("-", 35))
		fmt.Fprintln(&buf, "")

		for _, thing := range quotas {
			fmt.Fprintf(&buf, "%s: %v\n", thing.QuotaName, thing.QuotaValue)
		}
		fmt.Fprintln(&buf, "")
	}

	// Write to logs, but also output to stdout
	return buf.String()
}

// evalAwsQuota provides an interface to the command-line
func evalAwsQuota(cmd *cobra.Command, _ []string) error {
	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		return fmt.Errorf("failed to get cloud region flag: %w", err)
	}

	awsClient := &awsinternal.AWSConfiguration{
		Config: awsinternal.NewAwsV2(cloudRegionFlag),
	}
	quotaDetails, err := awsClient.GetServiceQuotas([]string{"eks", "vpc"})
	if err != nil {
		return fmt.Errorf("failed to get service quotas: %w", err)
	}

	messageHeader := fmt.Sprintf(
		"AWS Quota Health\nRegion: %s\n\nIf you encounter issues deploying your Kubefirst cluster, check these quotas and determine if you need to request a limit increase.",
		cloudRegionFlag,
	)
	result := printAwsQuotaWarning(messageHeader, quotaDetails)

	// Write to logs, but also output to stdout
	fmt.Println(reports.StyleMessage(result))
	return nil
}
