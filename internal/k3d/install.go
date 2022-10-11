package k3d

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// CreateK3dCluster create an k3d cluster
func CreateK3dCluster() error {
	log.Println("Create K3d cluster for local install")
	config := configs.ReadConfig()
	// I tried Terraform templates using: https://registry.terraform.io/providers/pvotal-tech/k3d/latest/docs/resources/cluster
	// it didn't worked as expected.

	// TODO: Create the Cluster
	// k3d cluster create kubefirst
	_, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "cluster", "create", viper.GetString("cluster-name"))
	if err != nil {
		log.Println("error creating gitlab namespace")
	}

	// TODO: Adds kubeconfig to the default place
	// k3d kubeconfig get kubefirst > ~/_tmp/k3d_config

	// TODO: Check install
	// kubectl cluster-info --kubeconfig ~/_tmp/k3d_config
	return nil
}
