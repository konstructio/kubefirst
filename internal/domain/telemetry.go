package domain

import (
	"errors"
	"os"

	"github.com/google/uuid"
	"github.com/kubefirst/kubefirst/pkg"
)

// Telemetry data that will be consumed by handlers and services
type Telemetry struct {
	MetricName    string
	Domain        string
	CLIVersion    string
	ClusterType   string
	ClusterId     string
	KubeFirstTeam string
}

// NewTelemetry is the Telemetry domain. When instantiating new Telemetries, we're able to validate domain specific
// values. In this way, domain, handlers and services can work in isolation, and Domain host business logic.
func NewTelemetry(metricName string, domain string, CLIVersion string) (Telemetry, error) {

	if len(metricName) == 0 {
		return Telemetry{}, errors.New("unable to create metric, missing metric name")
	}

	// scan for kubefirst_team env
	kubeFirstTeam := "false"
	if os.Getenv("kubefirst_team") == "true" {
		kubeFirstTeam = "true"
	}
	//initialize cluster id
	clusterId := uuid.New().String()

	// localhost installation doesn't provide hostedzone that are mainly used as domain in this context. In case a
	// hostedzone is not provided, we assume it's a localhost installation
	if len(domain) == 0 {

		return Telemetry{
			MetricName:    metricName,
			Domain:        clusterId,
			CLIVersion:    CLIVersion,
			KubeFirstTeam: kubeFirstTeam,
			ClusterType:   "mgmt",
			ClusterId:     clusterId,
		}, nil
	}

	domain, err := pkg.RemoveSubDomain(domain)
	if err != nil {
		return Telemetry{}, err
	}

	return Telemetry{
		MetricName:    metricName,
		Domain:        domain,
		CLIVersion:    CLIVersion,
		KubeFirstTeam: kubeFirstTeam,
		ClusterType:   "mgmt",
		ClusterId:     clusterId,
	}, nil
}
