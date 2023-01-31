package services

import (
	"github.com/segmentio/analytics-go"
)

// SegmentIoService hosts SegmentIO requirements
type SegmentIoService struct {
	SegmentIOClient analytics.Client
}

// NewSegmentIoService instantiate a new SegmentIO service.
func NewSegmentIoService(segmentIoClient analytics.Client) SegmentIoService {
	return SegmentIoService{
		SegmentIOClient: segmentIoClient,
	}
}

// EnqueueCountMetric use the service SegmentIO client that also has a http client to communicate with SegmentIO API.
func (service SegmentIoService) EnqueueCountMetric(metricName string, domain string, cliVersion string, kubeFirstTeam string, clusterId string, clusterType string) error {

	// Enqueues a track event that will be sent asynchronously.
	err := service.SegmentIOClient.Enqueue(analytics.Track{
		UserId: domain,
		Event:  metricName,
		Properties: analytics.NewProperties().
			Set("domain", domain).
			Set("cli_version", cliVersion).
			Set("cluster_id", clusterId).
			Set("cluster_type", clusterType).
			Set("kubefirst_team", kubeFirstTeam),
	})
	if err != nil {
		return err
	}

	return nil
}

// EnqueueIdentify implements SegmentIO Identify https://segment.com/docs/connections/sources/catalog/libraries/server/go/
func (service SegmentIoService) EnqueueIdentify(domain string) error {

	// Enqueues a Identify event that will be sent asynchronously.
	err := service.SegmentIOClient.Enqueue(analytics.Identify{
		UserId: domain,
		Type:   "identify",
	})
	if err != nil {
		return err
	}

	return nil
}
