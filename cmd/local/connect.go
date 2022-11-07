package local

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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

func runConnect(cmd *cobra.Command, args []string) error {
	fmt.Println("---debug---")
	fmt.Println("hi")
	//err := k8s.OpenGenericPortForward(false)
	//if err != nil {
	//	fmt.Println(err)
	//}
	fmt.Println("---debug---")

	config1 := configs.ReadConfig()
	kubeconfig := config1.KubeConfigPath
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	stopCh := make(chan struct{}, 1)
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})
	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed. For the port forwarding we just need
	// the output eventually
	//stream := genericclioptions.IOStreams{
	//	In:     os.Stdin,
	//	Out:    os.Stdout,
	//	ErrOut: os.Stderr,
	//}

	pfReq := k8s.PortForwardAPodRequest{
		RestConfig: cfg,
		Pod: v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "atlantis-0",
				Namespace: "atlantis",
			},
		},
		//Service: v1.Service{
		//	ObjectMeta: metav1.ObjectMeta{
		//		Name:      "argocd-server",
		//		Namespace: "argocd",
		//	},
		//},
		PodPort:   80,
		LocalPort: 4141,
		StopCh:    stopCh,
		ReadyCh:   readyCh,
	}
	//err = k8s.PortForwardAPod(pfReq)
	//if err != nil {
	//	panic(err)
	//}
	clientset, err := k8s.GetClientSet(false)
	err = k8s.PortForwardAKubefirstPod(clientset, pfReq)
	if err != nil {
		panic(err)
	}
	//err = k8s.PortForwardTESTING(pfReq)
	//if err != nil {
	//	panic(err)
	//}
	//
	<-readyCh

	println("Port forwarding is ready to get traffic. have fun!")
	//
	////select {
	////case <-readyCh:
	////	break
	////}

	return nil
}
