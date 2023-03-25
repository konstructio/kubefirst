package aws

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
	awsinternal "github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
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
		createAwsQuotaWarning.WriteString(fmt.Sprintf("\n"))
		for _, thing := range quotas {
			createAwsQuotaWarning.WriteString(fmt.Sprintf("%s: %v\n", thing.QuotaName, thing.QuotaValue))
		}
		createAwsQuotaWarning.WriteString(fmt.Sprintf("\n"))
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

	awsClient := &awsinternal.Conf
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
