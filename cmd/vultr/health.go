/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"

	"github.com/kubefirst/kubefirst/internal/vultr"
	"github.com/spf13/cobra"
)

// checkVultrCloudHealth returns relevant info regarding Vultr prior to executing
// certain commands
func checkVultrCloudHealth(cmd *cobra.Command, args []string) error {
	cloudRegion, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		return err
	}
	vultrConf := vultr.VultrConfiguration{
		Client:  vultr.NewVultr(),
		Context: context.Background(),
	}
	err = vultrConf.HealthCheck(cloudRegion)
	if err != nil {
		return err
	}

	return nil
}
