package segment

import (
	"fmt"
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
)

const (
	kubefirstClient string = "api"
)

func InitClient(clusterId, clusterType, gitProvider string) (telemetry.TelemetryEvent, error) {
	machineID, err := machineid.ID()
	if err != nil {
		return telemetry.TelemetryEvent{}, fmt.Errorf("failed to get machine ID: %w", err)
	}

	c := telemetry.TelemetryEvent{
		CliVersion:        configs.K1Version,
		CloudProvider:     k3d.CloudProvider,
		ClusterID:         clusterId,
		ClusterType:       clusterType,
		DomainName:        k3d.DomainName,
		GitProvider:       gitProvider,
		InstallMethod:     "kubefirst-launch",
		KubefirstClient:   kubefirstClient,
		KubefirstTeam:     os.Getenv("KUBEFIRST_TEAM"),
		KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
		MachineID:         machineID,
		ErrorMessage:      "",
		MetricName:        telemetry.ClusterInstallCompleted,
		UserId:            machineID,
	}

	return c, nil
}
