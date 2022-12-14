package k3d

import (
	"errors"
	"github.com/rs/zerolog/log"
	"os"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// CreateK3dCluster create an k3d cluster
func CreateK3dCluster() error {
	log.Info().Msg("creating K3d cluster...")
	config := configs.ReadConfig()
	// I tried Terraform templates using: https://registry.terraform.io/providers/pvotal-tech/k3d/latest/docs/resources/cluster
	// it didn't worked as expected.
	if !viper.GetBool("k3d.created") {
		// k3d cluster create kubefirst  --agents 3 --agents-memory 1024m  --registry-create k3d-kubefirst-registry:63630
		//_, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "cluster", "create", viper.GetString("cluster-name"),
		_, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "cluster", "create",
			viper.GetString("cluster-name"),
			"--agents", "3",
			"--agents-memory", "1024m",
			"--registry-create", "k3d-"+viper.GetString("cluster-name")+"-registry:63630",
			"--k3s-arg", `--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*`,
			"--k3s-arg", `--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*`,
			"--port", "80:80@loadbalancer",
			"--port", "443:443@loadbalancer")
		if err != nil {
			log.Info().Msg("error creating k3d cluster")
			return errors.New("error creating k3d cluster")
		}

		time.Sleep(20 * time.Second)
		// k3d kubeconfig get kubefirst > ~/_tmp/k3d_config
		///gitops/terraform/base/
		_ = os.MkdirAll(config.KubeConfigFolder, 0777)

		log.Info().Msgf(config.K3dPath, "kubeconfig", "get", viper.GetString("cluster-name"), ">", config.KubeConfigPath)
		out, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "kubeconfig", "get", viper.GetString("cluster-name"))
		if err != nil {
			return err
		}
		log.Info().Msg(config.KubeConfigPath)

		kubeConfig := []byte(out)
		err = os.WriteFile(config.KubeConfigPath, kubeConfig, 0644)
		if err != nil {
			log.Error().Err(err).Msg("error updating config")
			return errors.New("error updating config")
		}
		viper.Set("k3d.created", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("K3d Cluster already created")
	}
	return nil
}
