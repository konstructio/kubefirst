/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provision

import (
	"errors"
	"fmt"

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
)

func CreateMgmtCluster(gitAuth apiTypes.GitAuth, cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) error {
	clusterRecord, err := utilities.CreateClusterDefinitionRecordFromRaw(
		gitAuth,
		cliFlags,
		catalogApps,
	)

	if err != nil {
		return fmt.Errorf("error creating cluster definition record: %w", err)
	}

	clusterCreated, err := cluster.GetCluster(clusterRecord.ClusterName)
	if err != nil && !errors.Is(err, cluster.ErrNotFound) {
		log.Printf("error retrieving cluster %q: %v", clusterRecord.ClusterName, err)
		return fmt.Errorf("error retrieving cluster: %w", err)
	}

	if errors.Is(err, cluster.ErrNotFound) {
		if err := cluster.CreateCluster(*clusterRecord); err != nil {
			return fmt.Errorf("error creating cluster: %w", err)
		}
	}

	if clusterCreated.Status == "error" {
		cluster.ResetClusterProgress(clusterRecord.ClusterName)
		if err := cluster.CreateCluster(*clusterRecord); err != nil {
			return fmt.Errorf("error re-creating cluster after error state: %w", err)
		}
	}

	return nil
}
