package domain

import (
	"errors"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubefirst/kubefirst/pkg"
)

// Telemetry data that will be consumed by handlers and services
type Telemetry struct {
	MetricName string
	Domain     string
	CLIVersion string
	MachineId  string
}

// NewTelemetry is the Telemetry domain. When instantiating new Telemetries, we're able to validate domain specific
// values. In this way, domain, handlers and services can work in isolation, and Domain host business logic.
func NewTelemetry(metricName string, domain string, CLIVersion string) (Telemetry, error) {

	if len(metricName) == 0 {
		return Telemetry{}, errors.New("unable to create metric, missing metric name")
	}
	machineId, err := machineid.ID()
	if err != nil {
		return Telemetry{}, err
	}

	// localhost installation doesn't provide hostedzone that are mainly used as domain in this context. In case a
	// hostedzone is not provided, we assume it's a localhost installation
	if len(domain) == 0 {
		domain = machineId
		return Telemetry{
			MetricName: metricName,
			Domain:     domain,
			CLIVersion: CLIVersion,
			MachineId:  machineId,
		}, nil
	}

	// we store domain only, not subdomains
	domain, err = pkg.RemoveSubDomain(domain)
	if err != nil {
		return Telemetry{}, err
	}

	return Telemetry{
		MetricName: metricName,
		Domain:     domain,
		CLIVersion: CLIVersion,
		MachineId:  machineId,
	}, nil
}
