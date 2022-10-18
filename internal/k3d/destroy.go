package k3d

import (
	"errors"
	"log"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// DeleteK3dCluster delete a k3d cluster
func DeleteK3dCluster() error {
	log.Println("Delete K3d cluster ", viper.GetString("cluster-name"))
	config := configs.ReadConfig()
	_, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "cluster", "delete", viper.GetString("cluster-name"))
	if err != nil {
		log.Println("error deleting k3d cluster")
		return errors.New("error deleting k3d cluster")
	}
	time.Sleep(20 * time.Second)

	return nil
}
