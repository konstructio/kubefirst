/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3s

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // required for k8s authentication
)

type Service struct {
	cliFlags *types.CliFlags
}

func (s *Service) CreateCluster(ctx context.Context) error {
	progress.DisplayLogHints(20)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(ctx, s.cliFlags.InstallCatalogApps)
	if err != nil {
		return fmt.Errorf("validation of catalog apps failed: %w", err)
	}

	if !isValid {
		return errors.New("catalog validation failed")
	}

	err = ValidateProvidedFlags(s.cliFlags.GitProvider)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("provided flags validation failed: %w", err)
	}

	if err := provision.ManagementCluster(s.cliFlags, catalogApps); err != nil {
		return fmt.Errorf("failed to provision management cluster: %w", err)
	}

	return nil
}

func ValidateProvidedFlags(gitProvider string) error {
	progress.AddStep("Validate provided flags")

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
