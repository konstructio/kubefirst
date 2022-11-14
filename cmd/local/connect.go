package local

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	log.Println("opening Port Forward for console...")

	// every port forward has its own closing control. when a channel is closed, the port forward is also closed.
	vaultStopChannel := make(chan struct{}, 1)
	argoStopChannel := make(chan struct{}, 1)
	argoCDStopChannel := make(chan struct{}, 1)
	chartmuseumStopChannel := make(chan struct{}, 1)
	minioStopChannel := make(chan struct{}, 1)
	minioConsoleStopChannel := make(chan struct{}, 1)
	kubefirstConsoleStopChannel := make(chan struct{}, 1)
	AtlantisStopChannel := make(chan struct{}, 1)
	err := k8s.OpenPortForwardForLocalConnectWrapper(
		vaultStopChannel,
		argoStopChannel,
		argoCDStopChannel,
		chartmuseumStopChannel,
		minioStopChannel,
		minioConsoleStopChannel,
		kubefirstConsoleStopChannel,
		AtlantisStopChannel,
	)
	if err != nil {
		return err
	}

	// style UI with local URLs
	fmt.Println(reports.StyleMessage(reports.LocalConnectSummary()))

	log.Println("Kubefirst port forward done")
	log.Println("hanging port forwards until ctrl+c is called")

	// managing termination signal from the terminal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-sigs
		close(vaultStopChannel)
		close(argoStopChannel)
		close(argoCDStopChannel)
		close(chartmuseumStopChannel)
		close(minioStopChannel)
		close(minioConsoleStopChannel)
		close(kubefirstConsoleStopChannel)
		close(AtlantisStopChannel)
		log.Println("leaving port-forward command, port forwards are now closed")
		wg.Done()
	}()

	wg.Wait()

	return nil
}
