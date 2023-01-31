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
	ClusterId     string
	ClusterType   string
	KubeFirstTeam string
}

type Option func(f *Telemetry)

func WithClusterId(clusterId string) Option {
	return func(f *Telemetry) {
		f.ClusterId = clusterId
	}
}

func WithClusterType(clusterType string) Option {
	return func(f *Telemetry) {
		f.ClusterType = clusterType
	}
}

func WithKubeFirstTeam(kubeFirstTeam string) Option {
	return func(f *Telemetry) {
		f.KubeFirstTeam = kubeFirstTeam
	}
}

// NewTelemetry is the Telemetry domain. When instantiating new Telemetries, we're able to validate domain specific
// values. In this way, domain, handlers and services can work in isolation, and Domain host business logic.
func NewTelemetry(metricName string, domain string, CLIVersion string, opts ...Option) (Telemetry, error) {

	if len(metricName) == 0 {
		return Telemetry{}, errors.New("unable to create metric, missing metric name")
	}

	// scan for kubefirst_team env
	kubeFirstTeam := "false"
	if os.Getenv("kubefirst_team") == "true" {
		kubeFirstTeam = "true"
	}

	//initialize cluster type and id
	clusterId := uuid.New().String()
	clusterType := "mgmt"

	// localhost installation doesn't provide hostedzone that are mainly used as domain in this context. In case a
	// hostedzone is not provided, we assume it's a localhost installation
	if len(domain) == 0 {

		telemetry := Telemetry{
			MetricName:    metricName,
			Domain:        "",
			CLIVersion:    CLIVersion,
			KubeFirstTeam: kubeFirstTeam,
			ClusterType:   clusterType,
			ClusterId:     clusterId,
		}
		//populate with optional arguments
		for _, opt := range opts {
			opt(&telemetry)
		}
		telemetry.Domain = telemetry.ClusterId

		return telemetry, nil
	}

	domain, err := pkg.RemoveSubDomain(domain)
	if err != nil {
		return Telemetry{}, err
	}

	telemetry := Telemetry{
		MetricName:    metricName,
		Domain:        domain,
		CLIVersion:    CLIVersion,
		KubeFirstTeam: kubeFirstTeam,
		ClusterType:   clusterType,
		ClusterId:     clusterId,
	}
	//populate with optional arguments
	for _, opt := range opts {
		opt(&telemetry)
	}
	return telemetry, nil
}
