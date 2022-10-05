package handlers

import (
	"errors"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
)

// TelemetryHandler hosts handler requirements.
type TelemetryHandler struct {
	httpClient pkg.HTTPDoer
	service    services.SegmentIoService
}

// NewTelemetry instantiate a new Telemetry struct.
func NewTelemetry(httpClient pkg.HTTPDoer, service services.SegmentIoService) TelemetryHandler {
	return TelemetryHandler{
		httpClient: httpClient,
		service:    service,
	}
}

// SendCountMetric validate and handles the metric request to the metric service.
func (handler TelemetryHandler) SendCountMetric(metricName string, domain string, cliVersion string) error {

	if len(metricName) == 0 {
		return errors.New("unable to send metric, missing metric name")
	}

	err := handler.service.EnqueueCountMetric(metricName, domain, cliVersion)
	if err != nil {
		return err
	}

	return nil
}
