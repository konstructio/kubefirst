package local

import (
	"log"
	"time"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
)

func runPostLocal(cmd *cobra.Command, args []string) error {

	if !enableConsole {
		log.Println("not calling console, console flag is disabled")
		return nil
	}

	// open all port forwards, wait console ui be ready, and open console ui in the browser
	err := k8s.OpenPortForwardForKubeConConsole()
	if err != nil {
		log.Println(err)
	}

	time.Sleep(time.Millisecond * 2000)

	log.Println("Starting the presentation of console and api for the handoff screen")

	err = pkg.IsConsoleUIAvailable(pkg.LocalConsoleUI)
	if err != nil {
		log.Println(err)
	}
	err = pkg.OpenBrowser(pkg.LocalConsoleUI)
	if err != nil {
		return err
	}

	reports.LocalHandoffScreen(dryRun, silentMode)

	log.Println("Kubefirst Console available at: http://localhost:9094", silentMode)

	return nil
}
