package k3d

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// DeleteK3dCluster delete a k3d cluster
func DeleteK3dCluster() error {
	log.Info().Msgf("Delete K3d cluster %s", viper.GetString("cluster-name"))
	config := configs.ReadConfig()
	_, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "cluster", "delete", viper.GetString("cluster-name"))
	if err != nil {
		log.Info().Msg("error deleting k3d cluster")
		return errors.New("error deleting k3d cluster")
	}
	// todo: remove it?
	time.Sleep(20 * time.Second)

	volumeFolder := fmt.Sprintf("%s/minio-storage", config.K1FolderPath)
	os.RemoveAll(volumeFolder)

	return nil
}
