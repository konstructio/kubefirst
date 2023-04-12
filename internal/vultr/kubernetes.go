/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vultr/govultr/v3"
)

// GetKubernetesAssociatedVolumes returns block storage associated with a Vultr Kubernetes cluster
func (c *VultrConfiguration) GetKubernetesAssociatedBlockStorage(clusterName string, returnAll bool) ([]govultr.BlockStorage, error) {
	// Probably needs pagination
	allBlockStorage, _, _, err := c.Client.BlockStorage.List(c.Context, &govultr.ListOptions{})
	if err != nil {
		return []govultr.BlockStorage{}, err
	}

	if !returnAll {
		// Return only volumes associated with droplets part of the target cluster's node pool
		clusters, _, _, err := c.Client.Kubernetes.ListClusters(c.Context, &govultr.ListOptions{})
		if err != nil {
			return []govultr.BlockStorage{}, err
		}

		var clusterID string
		for _, cluster := range clusters {
			if cluster.Label == clusterName {
				clusterID = cluster.ID
			}
		}
		if clusterID == "" {
			return []govultr.BlockStorage{}, fmt.Errorf("could not find cluster ID for cluster name %s", clusterName)
		}

		cluster, _, err := c.Client.Kubernetes.GetCluster(c.Context, clusterID)
		if err != nil {
			return []govultr.BlockStorage{}, err
		}

		// Construct a slice of node IDs associated with a cluster's node pool
		nodeIDs := make([]string, 0)
		for _, pool := range cluster.NodePools {
			for _, inst := range pool.Nodes {
				nodeIDs = append(nodeIDs, inst.ID)
			}
		}

		// Return only block storage resources attached to a cluster's node pool droplets
		blockStorageToDelete := make([]govultr.BlockStorage, 0)
		for _, node := range nodeIDs {
			for _, blockStorage := range allBlockStorage {
				if blockStorage.AttachedToInstance == node {
					blockStorageToDelete = append(blockStorageToDelete, blockStorage)
				}
			}
		}

		return blockStorageToDelete, nil
	}

	if returnAll {
		// Return all block storage resources
		return allBlockStorage, nil
	}

	return []govultr.BlockStorage{}, nil
}

// DeleteBlockStorage iterates over target volumes and deletes them
func (c *VultrConfiguration) DeleteBlockStorage(blockStorage []govultr.BlockStorage) error {
	if len(blockStorage) == 0 {
		return fmt.Errorf("no block storage resources are available for deletion with the provided parameters")
	}

	for _, blst := range blockStorage {
		log.Info().Msg("removing block storage with name: " + blst.Label)
		err := c.Client.BlockStorage.Delete(c.Context, blst.ID)
		if err != nil {
			return err
		}
		log.Info().Msg("volume " + blst.ID + " deleted")
	}

	return nil
}
