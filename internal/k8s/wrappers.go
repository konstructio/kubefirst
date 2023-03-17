package k8s

import (
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// OpenPortForwardPodWrapper wrapper for PortForwardPod function. This functions make it easier to open and close port
// forward request. By providing the function parameters, the function will manage to create the port forward. The
// parameter for the stopChannel controls when the port forward must be closed.
//
// Example:
//
//	vaultStopChannel := make(chan struct{}, 1)
//	go func() {
//		OpenPortForwardWrapper(
//			pkg.VaultPodName,
//			pkg.VaultNamespace,
//			pkg.VaultPodPort,
//			pkg.VaultPodLocalPort,
//			vaultStopChannel)
//		wg.Done()
//	}()
func OpenPortForwardPodWrapper(clientset *kubernetes.Clientset, restConfig *rest.Config, podName, namespace string, podPort int, podLocalPort int, stopChannel chan struct{}) {

	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	// todo: constants for podName, PodPort and localPort, namespace

	portForwardRequest := PortForwardAPodRequest{
		RestConfig: restConfig,
		Pod: v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
		},
		PodPort:   podPort,
		LocalPort: podLocalPort,
		StopCh:    stopChannel,
		ReadyCh:   readyCh,
	}

	go func() {
		err := PortForwardPodWithRetry(clientset, portForwardRequest)
		if err != nil {
			log.Error().Err(err).Msg(err.Error())
		}
	}()

	select {
	case <-stopChannel:
		log.Info().Msg("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		log.Info().Msg("port forwarding is ready to get traffic")
	}

	log.Info().Msgf("Pod %q at namespace %q has port-forward accepting local connections at port %d\n", podName, namespace, podLocalPort)

}

func OpenPortForwardServiceWrapper(kubeconfigPath, kubeconfigClientPath, namespace, serviceName string, servicePort int, serviceLocalPort int, stopChannel chan struct{}) {

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	// todo: constants for podName, PodPort and localPort, namespace

	portForwardRequest := PortForwardAServiceRequest{
		RestConfig: kubeconfig,
		Service: v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
		},
		ServicePort: servicePort,
		LocalPort:   serviceLocalPort,
		StopCh:      stopChannel,
		ReadyCh:     readyCh,
	}

	clientset, err := GetClientSet(false, kubeconfigPath)

	go func() {
		// todo, i think we can use the RestConfig and remove the "kubectlClientPath"
		err = PortForwardService(clientset, kubeconfigPath, kubeconfigClientPath, portForwardRequest)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
	}()

	select {
	case <-stopChannel:
		log.Info().Msg("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		log.Info().Msg("port forwarding is ready to get traffic")
	}

	log.Info().Msgf("Service %q at namespace %q has port-forward accepting local connections at port %d\n", serviceName, namespace, serviceLocalPort)
	return
}
