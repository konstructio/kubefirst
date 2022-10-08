package k3d

import "log"

// CreateK3dCluster create an k3d cluster
func CreateK3dCluster() error {
	log.Println("Create K3d cluster for local install")
	// I tried Terraform templates using: https://registry.terraform.io/providers/pvotal-tech/k3d/latest/docs/resources/cluster
	// it didn't worked as expected.

	// TODO: Create the Cluster
	// k3d cluster create kubefirst

	// TODO: Adds kubeconfig to the default place
	// k3d kubeconfig get kubefirst > ~/_tmp/k3d_config

	// TODO: Check install
	// kubectl cluster-info --kubeconfig ~/_tmp/k3d_config
	return nil
}
