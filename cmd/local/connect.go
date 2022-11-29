package local

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func NewCommandConnect() *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "connect will open all Kubefirst services port forwards",
		Long: "connect opens all Kubefirst service ports for local connection, it makes the services available to" +
			"allow local request to the deployed services",
		RunE: runConnect,
	}

	return connectCmd
}

func runConnect(cmd *cobra.Command, args []string) error {
	log.Info().Msg("opening Port Forward for console...")

	// every port forward has its own closing control. when a channel is closed, the port forward is close.
	vaultStopChannel := make(chan struct{}, 1)
	argoStopChannel := make(chan struct{}, 1)
	argoCDStopChannel := make(chan struct{}, 1)
	chartmuseumStopChannel := make(chan struct{}, 1)
	minioStopChannel := make(chan struct{}, 1)
	minioConsoleStopChannel := make(chan struct{}, 1)
	kubefirstConsoleStopChannel := make(chan struct{}, 1)
	AtlantisStopChannel := make(chan struct{}, 1)
	MetaphorFrontendDevelopmentStopChannel := make(chan struct{}, 1)
	MetaphorGoDevelopmentStopChannel := make(chan struct{}, 1)
	MetaphorDevelopmentStopChannel := make(chan struct{}, 1)

	// guarantee it will close the port forwards even on a process kill
	defer func() {
		close(vaultStopChannel)
		close(argoStopChannel)
		close(argoCDStopChannel)
		close(chartmuseumStopChannel)
		close(minioStopChannel)
		close(minioConsoleStopChannel)
		close(kubefirstConsoleStopChannel)
		close(AtlantisStopChannel)
		close(MetaphorFrontendDevelopmentStopChannel)
		close(MetaphorGoDevelopmentStopChannel)
		close(MetaphorDevelopmentStopChannel)
		log.Info().Msg("leaving port-forward command, port forwards are now closed")
	}()

	err := k8s.OpenPortForwardForLocal(
		vaultStopChannel,
		argoStopChannel,
		argoCDStopChannel,
		chartmuseumStopChannel,
		minioStopChannel,
		minioConsoleStopChannel,
		kubefirstConsoleStopChannel,
		AtlantisStopChannel,
		MetaphorFrontendDevelopmentStopChannel,
		MetaphorGoDevelopmentStopChannel,
		MetaphorDevelopmentStopChannel,
	)
	if err != nil {
		return err
	}

	// style UI with local URLs
	fmt.Println(reports.StyleMessage(reports.LocalConnectSummary()))

	log.Info().Msg("Kubefirst port forward done")
	log.Info().Msg("hanging port forwards until ctrl+c is called")

	// managing termination signal from the terminal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-sigs
		wg.Done()
	}()

	wg.Wait()

	return nil
}
