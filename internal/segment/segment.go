package segment

import (
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/kubefirst/runtime/pkg/k3d"

	"github.com/segmentio/analytics-go"
)

const (
	kubefirstClient string = "api"
)

func InitClient(clusterId, clusterType, gitProvider string) *telemetry.SegmentClient {

	machineID, _ := machineid.ID()
	sc := analytics.New(telemetry.SegmentIOWriteKey)

	c := telemetry.SegmentClient{
		TelemetryEvent: telemetry.TelemetryEvent{
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
			UserId:            machineID,
			MetricName:        telemetry.ClusterInstallStarted,
		},
		Client: sc,
	}

	return &c
}
