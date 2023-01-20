package local

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
)

func runPostLocal(cmd *cobra.Command, args []string) error {

	if !enableConsole {
		log.Info().Msg("not calling console, console flag is disabled")
		return nil
	}

	log.Info().Msg("Starting the presentation of console and api for the handoff screen")

	err := pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURL)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURL)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	reports.LocalHandoffScreen(dryRun, silentMode)

	log.Info().Msgf("Kubefirst Console available at: %s", pkg.KubefirstConsoleLocalURLTLS)

	// managing termination signal from the terminal
	// todo: handle user inputs (q, ctrl+c, etc)
	//sigs := make(chan os.Signal, 1)
	//signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	//var wg sync.WaitGroup
	//wg.Add(1)
	//go func() {
	//	<-sigs
	//	wg.Done()
	//}()
	//wg.Wait()

	// todo: testing
	cancelContext()
	fmt.Println("---debug---")
	fmt.Println("context killed, waiting...")
	fmt.Println("---debug---")

	// force wait context kill
	time.Sleep(1 * time.Second)

	return nil
}
