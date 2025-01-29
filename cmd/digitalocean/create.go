/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"errors"
	"fmt"
	"os"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func createDigitalocean(cmd *cobra.Command, _ []string) error {
	cliFlags, err := utilities.GetFlags(cmd, "digitalocean")
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to get CLI flags: %w", err)
	}

	progress.DisplayLogHints(20)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if err != nil {
		return fmt.Errorf("catalog validation error: %w", err)
	}

	if !isValid {
		return errors.New("catalog did not pass a validation check")
	}

	err = ValidateProvidedFlags(cliFlags.GitProvider, cliFlags.DNSProvider)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate provided flags: %w", err)
	}

	return provision.ManagementCluster(cliFlags, catalogApps)
}

func ValidateProvidedFlags(gitProvider, dnsProvider string) error {
	progress.AddStep("Validate provided flags")

	// Validate required environment variables for dns provider
	if dnsProvider == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	for _, env := range []string{"DO_TOKEN", "DO_SPACES_KEY", "DO_SPACES_SECRET"} {
		if os.Getenv(env) == "" {
			return fmt.Errorf("your %q variable is unset - please set it before continuing", env)
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy")
		}
		log.Info().Msgf("%q %s", "github.com", key.Type())
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy")
		}
		log.Info().Msgf("%q %s", "gitlab.com", key.Type())
	}

	progress.CompleteStep("Validate provided flags")

	return nil
}
