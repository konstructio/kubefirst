/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/prechecks"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type mkcertOptions struct {
	application string
	namespace   string
}

func NewMkCertCommand() *cobra.Command {
	opts := &mkcertOptions{}

	cmd := &cobra.Command{
		Use:   "mkcert",
		Short: "create a single ssl certificate for a local application using mkcert (requires mkcert)",
		Long:  "create a single ssl certificate for a local application",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[PRECHECKS] Running prechecks")

			if err := prechecks.CommandExists("mkcert"); err != nil {
				return fmt.Errorf("mkcert is not installed, but is required when using k3d")
			}

			fmt.Println("[PRECHECKS] all prechecks passed - continuing")

			return nil
		},
		RunE: opts.runMkcert,
	}

	cmd.Flags().StringVar(&opts.application, "application", opts.application, "the name of the application (required)")
	cmd.MarkFlagRequired("application")

	cmd.Flags().StringVar(&opts.namespace, "namespace", opts.namespace, "the application namespace (required)")
	cmd.MarkFlagRequired("namespace")

	return cmd
}

func (o *mkcertOptions) runMkcert(cmd *cobra.Command, args []string) error {
	helpers.DisplayLogHints()

	flags := helpers.GetClusterStatusFlags()
	if !flags.SetupComplete {
		return fmt.Errorf("there doesn't appear to be an active k3d cluster")
	}

	config := k3d.GetConfig(
		viper.GetString("flags.cluster-name"),
		flags.GitProvider,
		viper.GetString(fmt.Sprintf("flags.%s-owner", flags.GitProvider)),
		flags.GitProtocol,
	)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	log.Infof("Generating certificate for %s.%s...", o.application, k3d.DomainName)

	if err := k3d.GenerateSingleTLSSecret(kcfg.Clientset, *config, o.application, o.namespace); err != nil {
		return fmt.Errorf("error generating certificate for %s/%s: %s", o.application, o.namespace, err)
	}

	log.Infof("Certificate generated. You can use it with an app by setting `tls.secretName: %s-tls` on a Traefik IngressRoute.", o.application)

	return nil
}
