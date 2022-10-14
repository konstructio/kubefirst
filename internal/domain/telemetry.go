package domain

import (
	"errors"
	"github.com/kubefirst/kubefirst/pkg"
)

// Telemetry data that will be consumed by handlers and services
type Telemetry struct {
	MetricName string
	Domain     string
	CLIVersion string
}

// NewTelemetry is the Telemetry domain. When instantiating new Telemetries, we're able to validate domain specific
// values. In this way, domain, handlers and services can work in isolation, and Domain host business logic.
func NewTelemetry(metricName string, domain string, CLIVersion string) (*Telemetry, error) {

	if len(metricName) == 0 {
		return nil, errors.New("unable to create metric, missing metric name")
	}

	domain, err := pkg.RemoveSubDomain(domain)
	if err != nil {
		return nil, err
	}

	return &Telemetry{
		MetricName: metricName,
		Domain:     domain,
		CLIVersion: CLIVersion,
	}, nil
}
