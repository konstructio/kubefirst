/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package segment

import (
	"fmt"

	"github.com/kubefirst/kubefirst/pkg"
	"github.com/segmentio/analytics-go"
)

func (c *SegmentClient) SendCountMetric(
	cliVersion string,
	cloudProvider string,
	clusterId string,
	clusterType string,
	domainName string,
	gitProvider string,
	kubefirstTeam string,
	metricName string,
) string {

	strippedDomainName, err := pkg.RemoveSubdomainV2(domainName)
	if err != nil {
		return "error stripping domain name from value"
	}

	if metricName == pkg.MetricInitStarted {
		err := c.Client.Enqueue(analytics.Identify{
			UserId: strippedDomainName,
			Type:   "identify",
		})
		if err != nil {
			return fmt.Sprintf("error sending identify to segment %s", err.Error())
		}
	}

	err = c.Client.Enqueue(analytics.Track{
		UserId: strippedDomainName,
		Event:  metricName,
		Properties: analytics.NewProperties().
			Set("cli_version", cliVersion).
			Set("cloud_provider", cloudProvider).
			Set("cluster_id", clusterId).
			Set("cluster_type", clusterType).
			Set("domain", strippedDomainName).
			Set("git_provider", gitProvider).
			Set("kubefirst_team", kubefirstTeam),
	})
	if err != nil {
		return fmt.Sprintf("error sending track to segment %s", err.Error())
	}

	return ""
}
