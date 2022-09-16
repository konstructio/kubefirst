/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"
	"strings"
	"syscall"

	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// k1ReadyCmd represents the argocdAppStatus command
// This command is used to check if cluster with basic tooling to allow install to proceed
var k1ReadyCmd = &cobra.Command{
	Use:   "k1-ready",
	Short: "Verify the status of key apps",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("argocdAppStatus called")

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}
		kPortForwardArgocd, err := k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
		defer func() {
			if kPortForwardArgocd != nil {
				_ = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			}
		}()
		if err != nil {
			//Port-forwarding may be already in play, if fails next commands will detect and fail as expected.
			log.Println("Error forwarding ports")

		}
		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, k1ReadyCmd skipped.")
			return nil
		}
		log.Println("argo forwarded called")
		argoCDUsername := viper.GetString("argocd.admin.username")
		argoCDPassword := viper.GetString("argocd.admin.password")
		token, err := argocd.GetArgoCDToken(argoCDUsername, argoCDPassword)
		if err != nil {
			return err
		}
		apps := strings.Fields("registry argocd atlantis cert-manager chartmuseum chartmuseum-components")
		for _, app := range apps {
			isAppSynched, err := argocd.IsAppSynched(token, app)
			if err != nil {
				return err
			}
			log.Println("App", app, "is synched:", isAppSynched)
			if !isAppSynched {
				log.Println("App", app, "is is not ready, synch status:", isAppSynched)
				return fmt.Errorf("app %s is is not ready, synch status: %v", app, isAppSynched)
			}
		}
		return nil
	},
}

func init() {
	actionCmd.AddCommand(k1ReadyCmd)
	currentCommand := k1ReadyCmd
	flagset.DefineGlobalFlags(currentCommand)

}
