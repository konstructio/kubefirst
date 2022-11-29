package local

import (
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
)

func runPostLocal(cmd *cobra.Command, args []string) error {

	if !enableConsole {
		log.Info().Msg("not calling console, console flag is disabled")
		return nil
	}

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

	log.Info().Msg("Starting the presentation of console and api for the handoff screen")

	err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURL)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURL)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	reports.LocalHandoffScreen(dryRun, silentMode)

	log.Info().Msgf("Kubefirst Console available at: http://localhost:9094", silentMode)

	// managing termination signal from the terminal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-sigs
		wg.Done()
	}()

	return nil
}
