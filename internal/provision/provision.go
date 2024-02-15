/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provision

import (
	apiTypes "github.com/kubefirst/kubefirst-api/pkg/types"
	runtimeTypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/internal/cluster"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/kubefirst/internal/types"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
)

func CreateMgmtCluster(gitAuth runtimeTypes.GitAuth, cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) {
	clusterRecord := utilities.CreateClusterDefinitionRecordFromRaw(
		gitAuth,
		cliFlags,
		catalogApps,
	)

	clusterCreated, err := cluster.GetCluster(clusterRecord.ClusterName)
	if err != nil {
		log.Info().Msg("cluster not found")
	}

	if !clusterCreated.InProgress {
		err = cluster.CreateCluster(clusterRecord)
		if err != nil {
			progress.Error(err.Error())
		}
	}

	if clusterCreated.Status == "error" {
		cluster.ResetClusterProgress(clusterRecord.ClusterName)
		cluster.CreateCluster(clusterRecord)
	}

	progress.StartProvisioning(clusterRecord.ClusterName)
}
