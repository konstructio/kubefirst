/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3s

import (
	"fmt"

	"github.com/rs/zerolog/log"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // required for k8s authentication
)

func ValidateProvidedFlags(gitProvider string) error {
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

	return nil
}
