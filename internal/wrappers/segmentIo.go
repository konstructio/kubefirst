package wrappers

import (
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/segmentio/analytics-go"
	"log"
)

// SendSegmentIoTelemetry is a wrapper function that instantiate SegmentIO handler, service, and sends a track activity to
// SegmentIO.
func SendSegmentIoTelemetry(hostedZone string, metricName string) error {
	// Instantiates a SegmentIO client to use send messages to the segment API.
	segmentIOClient := analytics.New(pkg.SegmentIOWriteKey)

	// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
	// to force flush in case there is still some pending message in the SegmentIO library queue.
	defer func(segmentIOClient analytics.Client) {
		err := segmentIOClient.Close()
		if err != nil {
			log.Println(err)
		}
	}(segmentIOClient)

	// validate telemetryDomain data
	telemetryDomain, err := domain.NewTelemetry(
		metricName,
		hostedZone,
		configs.K1Version,
	)
	if err != nil {
		return err
	}
	telemetryService := services.NewSegmentIoService(segmentIOClient)
	telemetryHandler := handlers.NewTelemetryHandler(telemetryService)

	err = telemetryHandler.SendCountMetric(telemetryDomain)
	if err != nil {
		return err
	}
	return nil
}
