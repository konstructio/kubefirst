/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/spf13/cobra"
)

type FlagConfig struct {
	Name     string
	FlagType string // "string" o "bool"
}

func getFlag(cmd *cobra.Command, config FlagConfig) (interface{}, error) {
	if config.FlagType == "string" {
		return cmd.Flags().GetString(config.Name)
	} else if config.FlagType == "bool" {
		return cmd.Flags().GetBool(config.Name)
	}
	return nil, fmt.Errorf("unknown flag type: %s", config.FlagType)
}

func validateFlags(cmd *cobra.Command, flagValues map[string]interface{}, FlagsToValidate []FlagConfig) error {
	for _, config := range FlagsToValidate {
		value, err := getFlag(cmd, config)
		if err != nil {
			return err
		}
		flagValues[config.Name] = value
	}
	return nil
}
