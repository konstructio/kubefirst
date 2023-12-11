package segment

import (
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg/k3d"
)

const (
	kubefirstClient string = "api"
)

func InitClient(clusterId, clusterType, gitProvider string) telemetry.TelemetryEvent {

	machineID, _ := machineid.ID()

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
		MetricName:        telemetry.ClusterInstallStarted,
		UserId:            machineID,
	}

	return c
}
