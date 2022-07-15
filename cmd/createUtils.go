package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/telemetry"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// todo: move it to internals/ArgoCD
func setArgocdCreds(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, setArgocdCreds skipped.")
		viper.Set("argocd.admin.password", "dry-run-not-real-pwd")
		viper.Set("argocd.admin.username", "dry-run-not-admin")
		viper.WriteConfig()
		return
	}

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

func sendStartedInstallTelemetry(dryRun bool) {
	metricName := "kubefirst.mgmt_cluster_install.started"
	if !dryRun {
		telemetry.SendTelemetry(viper.GetString("aws.hostedzonename"), metricName)
	} else {
		log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
	}
}

func sendCompleteInstallTelemetry(dryRun bool) {
	metricName := "kubefirst.mgmt_cluster_install.completed"
	if !dryRun {
		telemetry.SendTelemetry(viper.GetString("aws.hostedzonename"), metricName)
	} else {
		log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
	}
}

func waitArgoCDToBeReady(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, waitArgoCDToBeReady skipped.")
		return
	}
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/argocd")
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
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
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

func waitVaultToBeRunning(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
		return
	}
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/vault")
		if err != nil {
			log.Println("Waiting vault to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Println("vault namespace found, continuing")
			time.Sleep(25 * time.Second)
			break
		}
	}

	//! failing
	x = 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "get", "pods", "-l", "app.kubernetes.io/instance=vault")
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

func loopUntilPodIsReady() {

	x := 50
	url := "http://localhost:8200/v1/sys/health"
	for i := 0; i < x; i++ {
		log.Println("vault is not ready yet, sleeping and checking again")
		time.Sleep(10 * time.Second)

		req, _ := http.NewRequest("GET", url, nil)

		req.Header.Add("Content-Type", "application/json")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("error with http request Do, vault is not available", err)
			continue
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println("vault is availbale but the body is not what is expected ", err)
			continue
		}
		fmt.Println(string(body))

		var responseJson map[string]interface{}

		if err := json.Unmarshal(body, &responseJson); err != nil {
			log.Printf("vault is availbale but the body is not what is expected %s", err)
			continue
		}

		_, ok := responseJson["initialized"]
		if ok {
			log.Printf("vault is initialized and is in the expected state")
			return
		}
	}
	log.Panic("vault was never initialized")
}

type VaultInitResponse struct {
	Initialized                bool   `json:"initialized"`
	Sealed                     bool   `json:"sealed"`
	Standby                    bool   `json:"standby"`
	PerformanceStandby         bool   `json:"performance_standby"`
	ReplicationPerformanceMode string `json:"replication_performance_mode"`
	ReplicationDrMode          string `json:"replication_dr_mode"`
	ServerTimeUtc              int    `json:"server_time_utc"`
	Version                    string `json:"version"`
}

type VaultUnsealResponse struct {
	UnsealKeysB64         []interface{} `json:"unseal_keys_b64"`
	UnsealKeysHex         []interface{} `json:"unseal_keys_hex"`
	UnsealShares          int           `json:"unseal_shares"`
	UnsealThreshold       int           `json:"unseal_threshold"`
	RecoveryKeysBase64    []string      `json:"recovery_keys_base64"`
	RecoveryKeys          []string      `json:"recovery_keys"`
	RecoveryKeysShares    int           `json:"recovery_keys_shares"`
	RecoveryKeysThreshold int           `json:"recovery_keys_threshold"`
	RootToken             string        `json:"root_token"`
	Keys                  []string      `json:"keys"`
	KeysB64               []string      `json:"keys_base64"`
}

func initializeVaultAndAutoUnseal() {
	url := "http://127.0.0.1:8200/v1/sys/init"

	payload := strings.NewReader("{\n\t\"stored_shares\": 3,\n\t\"recovery_threshold\": 3,\n\t\"recovery_shares\": 5\n}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("error in Do http client request")
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	vaultResponse := VaultUnsealResponse{}
	json.Unmarshal(body, &vaultResponse)

	viper.Set("vault.token", vaultResponse.RootToken)
	viper.Set("vault.unseal-keys", vaultResponse)
	viper.WriteConfig()
}

func waitGitlabToBeReady(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
		return
	}
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/gitlab")
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
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "get", "pods", "-l", "app=webservice")
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

//Notify user in the STOUT and also logfile
func informUser(message string) {
	log.Println(message)
	progressPrinter.LogMessage(fmt.Sprintf("- %s", message))
}
