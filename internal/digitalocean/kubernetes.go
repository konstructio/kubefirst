/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"fmt"
	"time"

	"github.com/digitalocean/godo"
	"github.com/rs/zerolog/log"
)

// GetKubernetesAssociatedResources returns resources associated with a digitalocean Kubernetes cluster
func (c *DigitaloceanConfiguration) GetKubernetesAssociatedResources(clusterName string) (*godo.KubernetesAssociatedResources, error) {
	clusters, _, err := c.Client.Kubernetes.List(c.Context, &godo.ListOptions{})
	if err != nil {
		return &godo.KubernetesAssociatedResources{}, err
	}

	var clusterID string
	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			clusterID = cluster.ID
		}
	}
	if clusterID == "" {
		return &godo.KubernetesAssociatedResources{}, fmt.Errorf("could not find cluster ID for cluster name %s", err)
	}

	resources, _, err := c.Client.Kubernetes.ListAssociatedResourcesForDeletion(c.Context, clusterID)
	if err != nil {
		return &godo.KubernetesAssociatedResources{}, err
	}

	return resources, nil
}

// DeleteKubernetesClusterVolumes iterates over resource volumes and deletes them
func (c *DigitaloceanConfiguration) DeleteKubernetesClusterVolumes(resources *godo.KubernetesAssociatedResources) error {
	if len(resources.Volumes) == 0 {
		return fmt.Errorf("no volumes are available for deletion with the provided parameters")
	}

	for _, vol := range resources.Volumes {
		// Wait for volume to unattach
		for i := 0; i < 120; i++ {
			voldata, _, err := c.Client.Storage.GetVolume(c.Context, vol.ID)
			if err != nil {
				return err
			}
			if len(voldata.DropletIDs) != 0 {
				log.Info().Msgf("volume %s is still attached to droplet(s) - waiting...", vol.ID)
			}
			time.Sleep(time.Second * 1)
		}

		log.Info().Msg("removing volume with name: " + vol.Name)
		_, err := c.Client.Storage.DeleteVolume(c.Context, vol.ID)
		if err != nil {
			return err
		}
		log.Info().Msg("volume " + vol.ID + " deleted")
	}

	return nil
}
