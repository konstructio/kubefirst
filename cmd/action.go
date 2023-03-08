package cmd

import (
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/segment"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		segmentClient := &segment.Client

		segmentMsg := segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, "defdef", "mgmt", k3d.DomainName, "bitbucket", "true", pkg.MetricInitStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, "defdef", "mgmt", k3d.DomainName, "bitbucket", "true", pkg.MetricInitCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, "defdef", "mgmt", k3d.DomainName, "bitbucket", "true", pkg.MetricMgmtClusterInstallStarted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}
		segmentMsg = segmentClient.SendCountMetric(configs.K1Version, k3d.CloudProvider, "defdef", "mgmt", k3d.DomainName, "bitbucket", "true", pkg.MetricMgmtClusterInstallCompleted)
		if segmentMsg != "" {
			log.Info().Msg(segmentMsg)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
