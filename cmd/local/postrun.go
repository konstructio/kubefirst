package local

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kubefirst/kubefirst/configs"
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

	config := configs.ReadConfig()

	log.Info().Msg("storing certificates into application secrets namespace")
	if err := k8s.CreateSecretsFromCertificatesForLocalWrapper(config); err != nil {
		log.Error().Err(err).Msg("")
	}
	log.Info().Msg("storing certificates into application secrets namespace done")

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

	log.Info().Msgf("Kubefirst Console available at: http://localhost:9094", silentMode)
	_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/ingressroute.yaml", config.K1FolderPath))
	if err != nil {
		log.Printf("failed to create ingress route to argocd: %s", err)
	}

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
