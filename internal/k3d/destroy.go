package k3d

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/pkg"
)

// DeleteK3dCluster delete a k3d cluster
func DeleteK3dCluster(clusterName string, k1Dir string, k3dClient string) error {

	log.Info().Msgf("deleting k3d cluster %s", clusterName)
	_, _, err := pkg.ExecShellReturnStrings(k3dClient, "cluster", "delete", clusterName)
	if err != nil {
		log.Info().Msg("error deleting k3d cluster")
		return err
	}
	// todo: remove it?
	time.Sleep(20 * time.Second)

	volumeDir := fmt.Sprintf("%s/minio-storage", k1Dir)
	os.RemoveAll(volumeDir)

	return nil
}
