/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"errors"
	"fmt"
	"os"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/rs/zerolog/log"
)

type Service struct {
	cliFlags *types.CliFlags
}

func (s *Service) CreateCluster(_ context.Context) error {

	progress.DisplayLogHints(15)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(s.cliFlags.InstallCatalogApps)
	if err != nil {
		return fmt.Errorf("catalog apps validation failed: %w", err)
	}

	if !isValid {
		return errors.New("catalog validation failed")
	}

	err = ValidateProvidedFlags(s.cliFlags.GitProvider, s.cliFlags.DNSProvider)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("invalid provided flags: %w", err)
	}

	if err := provision.ManagementCluster(s.cliFlags, catalogApps); err != nil {
		return fmt.Errorf("failed to provision management cluster: %w", err)
	}

	return nil
}

func ValidateProvidedFlags(gitProvider, dnsProvider string) error {
	progress.AddStep("Validate provided flags")

	if os.Getenv("VULTR_API_KEY") == "" {
		return fmt.Errorf("your VULTR_API_KEY variable is unset - please set it before continuing")
	}

	if dnsProvider == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("%q %s", "github.com", key.Type())
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("%q %s", "gitlab.com", key.Type())
	}

	progress.CompleteStep("Validate provided flags")
	return nil
}
