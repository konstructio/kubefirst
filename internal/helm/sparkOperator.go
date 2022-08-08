package helm

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// InstallSparkOperator - Install Spark Operator
func InstallSparkOperator(dryRun bool) {
	config := configs.ReadConfig()
	if !viper.GetBool("create.spark.helm") {
		if dryRun {
			log.Printf("[#99] Dry-run mode, InstallSparkOperator skipped.")
			return
		}
		// ! commenting out until a clean execution is necessary // create namespace
		_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "add", "spark-operator", "https://googlecloudplatform.github.io/spark-on-k8s-operator")
		if err != nil {
			log.Panicf("error: could not run helm repo add %s", err)
		}

		_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "update")
		if err != nil {
			log.Panicf("error: could not helm repo update %s", err)
		}

		_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "install", "spark-op-release", "spark-operator/spark-operator", "--namespace", "spark-operator", "--create-namespace", "--wait")
		if err != nil {
			log.Panicf("error: could not helm install spark-operator command %s", err)
		}

		viper.Set("create.spark.helm", true)
		err = viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}
	}
}

// UninstallSparkOperator - Remove spark operator
func UninstallSparkOperator(dryRun bool) {
	config := configs.ReadConfig()
	if viper.GetBool("create.spark.helm") {
		if dryRun {
			log.Printf("[#99] Dry-run mode, UninstallSparkOperator skipped.")
			return
		}

		_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "uninstall", "spark-op-release", "spark-operator/spark-operator", "--namespace", "spark-operator", "--wait")
		if err != nil {
			log.Panicf("error: could not helm install spark-operator command %s", err)
		}

		viper.Set("create.spark.helm", false)
		err = viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}
	}
}
