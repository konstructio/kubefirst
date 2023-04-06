/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package configs

// code from: https://github.com/carolynvs/stingoftheviper

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// it follows Viper precedence:
// 1. explicit call to Set
// 2. flag
// 3. env
// 4. config
// 5. key/value store
// 6. default

// the input file that is able to provide is called kubefirst.yaml, and should be at the root folder, where the user has
// it's Kubefirst binary. Following the flag name convention is enough to have a Kubefirst config file.

// FLAGS VARIABLE
// example loading values from flags:
// go run . command-name --admin-email user@example.com

// ENVIRONMENT VARIABLE
// example loading environment variables:
// export KUBEFIRST_CLOUD=k3d
// command line commands loads the values from the environment variable and override the command flag.

// YAML
// example of a YAML Kubefirst file:
// admin-email: user@example.com
// cloud: k3d
// command line commands loads the value from the kubefirst.yaml and override the command flags.

const (
	// The name of our config file, without the file extension because viper supports many different config file languages.
	defaultConfigFilename = "kubefirst-config"

	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --number is bound to STING_NUMBER.
	envPrefix = "KUBEFIRST"
)

func InitializeViperConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(defaultConfigFilename)
	v.SetConfigType("yaml")

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(".")

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	//if err := v.ReadInConfig(); err != nil {
	// It's okay if there isn't a config file
	//if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
	//	return err
	//}
	//return err
	//}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
