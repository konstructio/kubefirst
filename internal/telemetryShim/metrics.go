/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package telemetryShim

import (
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/rs/zerolog/log"
)

// Transmit sends a metric via Segment
func Transmit(useTelemetry bool, segmentClient *segment.SegmentClient, metricName string, errorMessage string) {
	segmentMsg := segmentClient.SendCountMetric(metricName, errorMessage)
	if segmentMsg != "" {
		log.Info().Msg(segmentMsg)
	}
}
