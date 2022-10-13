package k3d

import (
	"errors"
	"log"
	"os"
	"time"

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

	// k3d registry create kubefirst-registry --port 63630
	// _, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "registry", "create", "kubefirst-registry", "--port", "63630")
	// if err != nil {
	// 	log.Println("error creating k3d registry")
	// 	return errors.New("error creating k3d registry")
	// }

	// k3d cluster create kubefirst  --agents 3 --agents-memory 1024m  --registry-create k3d-kubefirst-registry:63630
	_, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "cluster", "create", viper.GetString("cluster-name"),
		"--agents", "3", "--agents-memory", "1024m", "--registry-create", "k3d-"+viper.GetString("cluster-name")+"-registry:63630")
	if err != nil {
		log.Println("error creating k3d cluster")
		return errors.New("error creating k3d cluster")
	}

	time.Sleep(20 * time.Second)
	// k3d kubeconfig get kubefirst > ~/_tmp/k3d_config
	///gitops/terraform/base/
	_ = os.MkdirAll(config.KubeConfigFolder, 0777)

	log.Println(config.K3dPath, "kubeconfig", "get", viper.GetString("cluster-name"), ">", config.KubeConfigPath)
	out, _, err := pkg.ExecShellReturnStrings(config.K3dPath, "kubeconfig", "get", viper.GetString("cluster-name"))
	log.Println(config.KubeConfigPath)

	kubeConfig := []byte(out)
	err = os.WriteFile(config.KubeConfigPath, kubeConfig, 0644)
	if err != nil {
		log.Println("error updating config:", err)
		return errors.New("error updating config")
	}

	// kubectl cluster-info --kubeconfig ~/_tmp/k3d_config
	//_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "cluster-info", "--kubeconfig", config.KubeConfigPath)
	//if err != nil {
	//		log.Println("error checking k3d cluster")
	//}
	return nil
}
