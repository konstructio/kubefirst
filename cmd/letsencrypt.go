/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/runtime/pkg/certificates"
	"github.com/spf13/cobra"
)

var (
	// Certificate check
	domainNameFlag string
)

func LetsEncryptCommand() *cobra.Command {
	letsEncryptCommand := &cobra.Command{
		Use:   "letsencrypt",
		Short: "interact with letsencrypt certificates for a domain",
		Long:  "interact with letsencrypt certificates for a domain",
	}

	// wire up new commands
	letsEncryptCommand.AddCommand(status())

	return letsEncryptCommand
}

func status() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:              "status",
		Short:            "check the usage statistics for a letsencrypt certificate",
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			err := certificates.CheckCertificateUsage(domainNameFlag)
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	// todo review defaults and update descriptions
	statusCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the domain to check certificates for (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	statusCmd.MarkFlagRequired("domain-name")

	return statusCmd
}
