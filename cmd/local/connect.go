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

// NewCommandConnect open local port to enable connection between your local computer, to the cluster applications.
func NewCommandConnect() *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "local connect command enable port forwards",
		Long: `local connect command enable local tunnels to enable local connection to the cluster application. Check 
logs for details.`,
		RunE: runConnect,
	}

	return connectCmd
}

// runConnect opens port forwards for the available Kubefirst applications.
func runConnect(cmd *cobra.Command, args []string) error {

	log.Println("opening Port Forward for console...")

	err := k8s.OpenPortForwardForKubeConConsole()
	if err != nil {
		return err
	}

	// style UI with local URLs
	fmt.Println(reports.StyleMessage(reports.LocalConnectSummary()))

	// managing termination signal from the terminal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-sigs
		log.Println("leaving port-forward command, port forwards are now closed")
		wg.Done()
	}()
	wg.Wait()

	log.Println("Kubefirst port forward done")
	log.Println("hanging port forwards until ctrl+c is called")

	return nil
}
