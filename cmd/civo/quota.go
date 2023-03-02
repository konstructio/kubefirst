package civo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/civo/civogo"
	"github.com/fatih/color"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	// The threshold at which civo quotas will trigger a warning
	quotaObjectThresholdWarning = 80
	// The threshold at which civo quotas will trigger a critical warning
	quotaObjectThresholdCritical = 90
	// The link to request a quota limit increase within Civo
	civoQuotaIncreaseLink = "https://dashboard.civo.com/quota/edit"
)

// checkFields asserts which limits should be checked against the civo quota
// These fields are the json values of the Quota struct
// https://github.com/civo/civogo/blob/master/quota.go#L9
// All values used here must be of type int, excluding the string fields
var checkFields map[string]string = map[string]string{
	"cpu_core_limit":            "cpu_core_usage",
	"database_count_limit":      "database_count_usage",
	"database_cpu_core_limit":   "database_cpu_core_usage",
	"database_ram_mb_limit":     "database_ram_mb_usage",
	"database_disk_gb_limit":    "database_disk_gb_usage",
	"disk_gb_limit":             "disk_gb_usage",
	"disk_volume_count_limit":   "disk_volume_count_usage",
	"instance_count_limit":      "instance_count_usage",
	"loadbalancer_count_limit":  "loadbalancer_count_usage",
	"network_count_limit":       "network_count_usage",
	"objectstore_gb_limit":      "objectstore_gb_usage",
	"port_count_limit":          "port_count_usage",
	"public_ip_address_limit":   "public_ip_address_usage",
	"ram_mb_limit":              "ram_mb_usage",
	"security_group_limit":      "security_group_usage",
	"security_group_rule_limit": "security_group_rule_usage",
	"subnet_count_limit":        "subnet_count_usage",
}

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// quotaFormattedOutput provides individual items returned from quota health check
type quotaFormattedOutput struct {
	actualFieldTitle string
	actualFieldValue float64
	limitValue       float64
}

// returnCivoQuotaEvaluation fetches quota from civo and compares limits to usage
func returnCivoQuotaEvaluation(cloudRegion string, showAll bool) (string, int, int, error) {
	// Fetch quota from civo
	client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), cloudRegion)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", 0, 0, err
	}

	quota, err := client.GetQuota()
	if err != nil {
		log.Info().Msgf("failed to fetch civo quota: %s", err)
		return "", 0, 0, err

	}

	// Container for quota response as a map
	var quotaMap map[string]interface{}

	// Marshal quota and unmarshal into map
	quotaJSON, err := json.Marshal(quota)
	if err != nil {
		log.Info().Msgf("failed to marshal civo quota struct: %s", err)
		return "", 0, 0, err
	}
	err = json.Unmarshal(quotaJSON, &quotaMap)
	if err != nil {
		log.Info().Msgf("failed to unmarshal civo quota struct: %s", err)
		return "", 0, 0, err
	}

	// Compare actual to limit and warn against threshold
	output := make([]string, 0)
	quotaFailures := 0
	quotaWarnings := 0

	for limitField, actualField := range checkFields {
		// Calculate the percent of a given limit that has been used
		percentCalc := math.Round(
			quotaMap[actualField].(float64) / quotaMap[limitField].(float64) * 100,
		)
		percentExpr := fmt.Sprintf("%v%%", percentCalc)
		checkObj := quotaFormattedOutput{
			actualFieldTitle: actualField,
			actualFieldValue: quotaMap[actualField].(float64),
			limitValue:       quotaMap[limitField].(float64),
		}

		switch {
		case percentCalc > quotaObjectThresholdWarning && percentCalc < quotaObjectThresholdCritical:
			quotaWarnings += 1
			outputFormat := checkObj.formatQuotaOutput(yellow(percentExpr))
			output = append(output, outputFormat)
		case percentCalc > quotaObjectThresholdCritical:
			quotaFailures += 1
			outputFormat := checkObj.formatQuotaOutput(red(percentExpr))
			output = append(output, outputFormat)
		default:
			if showAll {
				outputFormat := checkObj.formatQuotaOutput(green(percentExpr))
				output = append(output, outputFormat)
			}
		}
	}

	// Parse the entire message
	var messageHeader = fmt.Sprintf("Civo Quota Health\nRegion: %s\n\nNote that if any of these are approaching their limits, you may want to increase them.", cloudRegion)
	sort.Strings(output)
	result := printCivoQuotaWarning(messageHeader, output)

	return result, quotaFailures, quotaWarnings, nil
}

// formatQuotaOutput returns a formatted string representation of a specific quota comparison
func (q quotaFormattedOutput) formatQuotaOutput(usageExpression string) string {
	return fmt.Sprintf("%s - %v used of %v [%v]",
		q.actualFieldTitle,
		q.actualFieldValue,
		q.limitValue,
		usageExpression,
	)
}

// printCivoQuotaWarning provides visual output detailing quota health
func printCivoQuotaWarning(messageHeader string, output []string) string {
	var createCivoQuotaWarning bytes.Buffer
	createCivoQuotaWarning.WriteString(strings.Repeat("-", 70))
	createCivoQuotaWarning.WriteString(fmt.Sprintf("\n%s\n", messageHeader))
	createCivoQuotaWarning.WriteString(strings.Repeat("-", 70))
	createCivoQuotaWarning.WriteString("\n")
	for _, result := range output {
		createCivoQuotaWarning.WriteString(fmt.Sprintf("%s\n", result))
	}
	if len(output) == 0 {
		createCivoQuotaWarning.WriteString("All quotas are healthy. To show all quotas regardless, run `kubefirst civo quota --show-all`\n")
	}
	createCivoQuotaWarning.WriteString("\nIf you encounter any errors while working with Civo, request a limit increase for your account before retrying.\n\n")
	createCivoQuotaWarning.WriteString(civoQuotaIncreaseLink)

	// Write to logs, but also output to stdout
	return createCivoQuotaWarning.String()

}

// evalCivoQuota provides an interface to the command-line
func evalCivoQuota(cmd *cobra.Command, args []string) error {
	civoToken := os.Getenv("CIVO_TOKEN")
	if len(civoToken) == 0 {
		return errors.New("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security and set CIVO_TOKEN.\n")
	}

	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		return err
	}

	quotaShowAllFlag, err := cmd.Flags().GetBool("show-all")
	if err != nil {
		return err
	}

	message, _, _, err := returnCivoQuotaEvaluation(cloudRegionFlag, quotaShowAllFlag)
	if err != nil {
		return err
	}

	// Write to logs, but also output to stdout
	fmt.Println(reports.StyleMessage(message))

	return nil
}
