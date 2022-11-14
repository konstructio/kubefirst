package local

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"time"
)

func NewCommandConnect() *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "",
		Long:  "",
		RunE:  runConnect,
	}

	return connectCmd
}

func portForward(podName string, namespace string, podPort int, podLocalPort int, stopChannel chan struct{}) {

	config1 := configs.ReadConfig()
	kubeconfig := config1.KubeConfigPath
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err)
	}

	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	// todo: constants for podName, PodPort and localPort, namespace
	portForwardRequest := k8s.PortForwardAPodRequest{
		RestConfig: cfg,
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

	clientset, err := k8s.GetClientSet(false)

	go func() {
		fmt.Println("---debug---")
		fmt.Println("opening port forward")
		fmt.Println("---debug---")

		err = k8s.PortForwardAKubefirstPod(clientset, portForwardRequest)
		if err != nil {
			log.Println(err)
		}
	}()

	fmt.Println("Port forwarding is ready to get traffic. have fun!")

	select {
	case <-stopChannel:
		fmt.Println("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		fmt.Println("accepting connections")
	}

	fmt.Println("---debug---")
	fmt.Println("waiting...")
	fmt.Println("---debug---")

	fmt.Printf("Pod %q at namespace %q has port forward accepting connections at port %d\n", podName, namespace, podLocalPort)
	<-stopChannel

	return
}

func runConnect(cmd *cobra.Command, args []string) error {
	//err := k8s.OpenGenericPortForward(false)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//
	//config1 := configs.ReadConfig()
	//kubeconfig := config1.KubeConfigPath
	//cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	//if err != nil {
	//	panic(err)
	//}
	//
	//stopCh := make(chan struct{}, 1)
	//// readyCh communicate when the port forward is ready to get traffic
	//readyCh := make(chan struct{})
	//// stream is used to tell the port forwarder where to place its output or
	//// where to expect input if needed. For the port forwarding we just need
	//// the output eventually
	////stream := genericclioptions.IOStreams{
	////	In:     os.Stdin,
	////	Out:    os.Stdout,
	////	ErrOut: os.Stderr,
	////}
	//
	//// todo: constants for podName, PodPort and localPort, namespace
	//portForwardRequest := k8s.PortForwardAPodRequest{
	//	RestConfig: cfg,
	//	Pod: v1.Pod{
	//		ObjectMeta: metav1.ObjectMeta{
	//			Name:      "vault-0",
	//			Namespace: "vault",
	//		},
	//	},
	//	//Service: v1.Service{
	//	//	ObjectMeta: metav1.ObjectMeta{
	//	//		Name:      "argocd-server",
	//	//		Namespace: "argocd",
	//	//	},
	//	//},
	//	PodPort:   8200,
	//	LocalPort: 8200,
	//	StopCh:    stopCh,
	//	ReadyCh:   readyCh,
	//}
	////err = k8s.PortForwardAPod(pfReq)
	////if err != nil {
	////	panic(err)
	////}
	//clientset, err := k8s.GetClientSet(false)
	//
	//go func() {
	//	fmt.Println("---debug---")
	//	fmt.Println("opening port forward")
	//	fmt.Println("---debug---")
	//
	//	err = k8s.PortForwardAKubefirstPod(clientset, portForwardRequest)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()
	//
	////err = k8s.PortForwardTESTING(pfReq)
	////if err != nil {
	////	panic(err)
	////}
	////
	//
	//fmt.Println("Port forwarding is ready to get traffic. have fun!")
	////
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	fmt.Println("---debug---")
	//	fmt.Println("stop call")
	//	fmt.Println("---debug---")
	//	close(stopCh)
	//
	//}()
	//
	//select {
	//case <-stopCh:
	//	fmt.Println("leaving...")
	//	close(stopCh)
	//	close(readyCh)
	//	break
	//case <-readyCh:
	//	fmt.Println("accepting connections")
	//}
	//
	//fmt.Println("---debug---")
	//fmt.Println("waiting...")
	//fmt.Println("---debug---")
	//
	//<-stopCh

	vaultStopChannel := make(chan struct{}, 1)
	go func() {
		portForward(VaultPodName, VaultNamespace, VaultPodPort, VaultPorLocalPort, vaultStopChannel)
	}()

	time.Sleep(10 * time.Second)
	fmt.Println("---debug---")
	fmt.Println("stop call")
	fmt.Println("---debug---")
	defer func() {
		close(vaultStopChannel)
	}()

	return nil
}

const (
	VaultPodName      = "vault-0"
	VaultNamespace    = "vault"
	VaultPodPort      = 8200
	VaultPorLocalPort = 8200
)
