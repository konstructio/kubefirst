package handlers

import (
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/services"
)

// TelemetryHandler hosts handler requirements.
type TelemetryHandler struct {
	service services.SegmentIoService
}

// NewTelemetryHandler instantiate a new Telemetry handler.
func NewTelemetryHandler(service services.SegmentIoService) TelemetryHandler {
	return TelemetryHandler{
		service: service,
	}
}

// SendCountMetric validate and handles the metric request to the metric service.
func (handler TelemetryHandler) SendCountMetric(telemetry *domain.Telemetry) error {

	err := handler.service.EnqueueCountMetric(
		telemetry.MetricName,
		telemetry.Domain,
		telemetry.CLIVersion,
	)
	if err != nil {
		return err
	}

	return nil
}
