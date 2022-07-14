package cmd

import (
	"github.com/kubefirst/kubefirst/internal/telemetry"
	"github.com/spf13/viper"
	"github.com/kubefirst/kubefirst/configs"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"os/exec"
	"time"
)	



// todo: move it to internals/ArgoCD
func setArgocdCreds() {
	cfg := configs.ReadConfig()
	config, err := clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	argocdSecretClient = clientset.CoreV1().Secrets("argocd")

	argocdPassword := getSecretValue(argocdSecretClient, "argocd-initial-admin-secret", "password")

	viper.Set("argocd.admin.password", argocdPassword)
	viper.Set("argocd.admin.username", "admin")
	viper.WriteConfig()
}

func sendStartedInstallTelemetry(dryRun bool){
	metricName := "kubefirst.mgmt_cluster_install.started"
	if !dryRun {
		telemetry.SendTelemetry( viper.GetString("aws.hostedzonename"), metricName)
	} else {
		log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
	}
}

func sendCompleteInstallTelemetry(dryRun bool){
	metricName := "kubefirst.mgmt_cluster_install.completed"
	if !dryRun {
		telemetry.SendTelemetry(viper.GetString("aws.hostedzonename"), metricName)
	} else {
		log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
	}
}

func waitArgoCDToBeReady(){
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/argocd")
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println("Waiting argocd to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("argocd namespace found, continuing")
			time.Sleep(5 * time.Second)
			break
		}
	}
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println("Waiting for argocd pods to create, checking in 10 seconds")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("argocd pods found, continuing")
			time.Sleep(15 * time.Second)
			break
		}
	}
}

func waitVaultToBeInitialized() {
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/vault")
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println("Waiting vault to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("vault namespace found, continuing")
			time.Sleep(25 * time.Second)
			break
		}
	}
	x = 50
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "get", "pods", "-l", "vault-initialized=true")
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println("Waiting vault pods to create")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("vault pods found, continuing")
			time.Sleep(15 * time.Second)
			break
		}
	}
}

func waitGitlabToBeReady() {
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/gitlab")
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println("Waiting gitlab namespace to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("gitlab namespace found, continuing")
			time.Sleep(5 * time.Second)
			break
		}
	}
	x = 50
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "get", "pods", "-l", "app=webservice")
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println("Waiting gitlab pods to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("gitlab pods found, continuing")
			time.Sleep(15 * time.Second)
			break
		}
	}

}
