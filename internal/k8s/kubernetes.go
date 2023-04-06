/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var GitlabSecretClient coreV1Types.SecretInterface

type PatchJson struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

func GetSecretValue(k8sClient coreV1Types.SecretInterface, secretName, key string) string {
	secret, err := k8sClient.Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("error getting key: %s from secret: %s", key, secretName)
	}
	return string(secret.Data[key])
}

// GetClientSet - Get reference to k8s credentials to use APIS
func GetClientSet(dryRun bool, kubeconfigPath string) (*kubernetes.Clientset, error) {
	if dryRun {
		log.Info().Msgf("[#99] Dry-run mode, GetClientSet skipped.")
		return nil, nil
	}

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Error getting kubeconfig")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error().Err(err).Msg("Error getting clientset")
		return clientset, err
	}

	return clientset, nil
}

// GetClientConfig returns a rest.Config object for working with the Kubernetes
// API
func GetClientConfig(dryRun bool, kubeconfigPath string) (*rest.Config, error) {
	if dryRun {
		log.Info().Msgf("[#99] Dry-run mode, GetClientConfig skipped.")
		return nil, nil
	}

	clientconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Error getting kubeconfig")
		return nil, err
	}

	return clientconfig, nil
}

// deprecated
// PortForward - opens port-forward to services
func PortForward(dryRun bool, filter, kubeconfigPath, kubectlClientPath, namespace, ports string) (*exec.Cmd, error) {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, K8sPortForward skipped.")
		return nil, nil
	}
	// config := configs.ReadConfig()

	var kPortForwardOutb, kPortForwardErrb bytes.Buffer
	kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "port-forward", filter, ports)
	kPortForward.Stdout = &kPortForwardOutb
	kPortForward.Stderr = &kPortForwardErrb
	err := kPortForward.Start()

	// make port forward port available for log
	log.Info().Msgf("kubectl port-forward started for (%s) available at http://localhost:%s", filter, strings.Split(ports, ":")[0])
	//defer kPortForwardVault.Process.Signal(syscall.SIGTERM)

	//Please, don't remove this sleep, pf takes a while to be ready to search calls.
	//So, if next command is called to curl this address it will get connection refused.
	//this sleep protects that.
	//Please, don't remove this comment either.
	time.Sleep(time.Second * 5)
	log.Info().Msgf("%s %s %s %s %s %s %s %s", kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "port-forward", filter, ports)
	if err != nil {
		// If it doesn't error, we kinda don't care much.
		log.Info().Msgf("Commad Execution STDOUT: %s", kPortForwardOutb.String())
		log.Error().Err(err).Msgf("Commad Execution STDERR: %s", kPortForwardErrb.String())
		log.Error().Err(err).Msgf("$error: failed to port-forward to %s in main thread", filter)
		return kPortForward, err
	}

	return kPortForward, nil
}

func WaitForNamespaceandPods(dryRun bool, kubeconfigPath, kubectlClientPath, namespace, podLabel string) {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, WaitForNamespaceandPods skipped")
		return
	}
	if !viper.GetBool("create.softserve.ready") {
		x := 50
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "get", fmt.Sprintf("namespace/%s", namespace))
			if err != nil {
				log.Info().Msg(fmt.Sprintf("waiting for %s namespace to create ", namespace))
				time.Sleep(10 * time.Second)
			} else {
				log.Info().Msg(fmt.Sprintf("namespace %s found, continuing", namespace))
				time.Sleep(10 * time.Second)
				i = 51
			}
		}
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "get", "pods", "-l", podLabel)
			if err != nil {
				log.Info().Msg(fmt.Sprintf("waiting for %s pods to create ", namespace))
				time.Sleep(10 * time.Second)
			} else {
				log.Info().Msg(fmt.Sprintf("%s pods found, continuing", namespace))
				time.Sleep(10 * time.Second)
				break
			}
		}
		viper.Set("create.softserve.ready", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("soft-serve is ready, skipping")
	}
}
