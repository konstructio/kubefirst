package flagset

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const ENV = 1
const FLAG = 0
const NONE = 99

// ReadConfigString - Read a Cobra-CLI flag or env variable of type string, precedence rule (Flags first, envVar second)
// default value is ""
// Current version doesn't have error, but we may expect some as the function evolves.
func ReadConfigString(cmd *cobra.Command, flag string) (string, error) {
	if DefineSource(cmd, flag) == ENV {
		return os.Getenv(GetFlagVarName(flag)), nil
	}
	value, err := cmd.Flags().GetString(flag)
	return value, err
}

func ReadConfigStringSlice(cmd *cobra.Command, flag string) ([]string, error) {
	value, err := cmd.Flags().GetStringSlice(flag)
	return value, err
}

// ReadConfigBool - Read a Cobra-CLI flag or env variable of type bool, precedence rule (Flags first, envVar second)
// default value is ""
// Current version doesn't have error, but we may expect some as the function evolves.
func ReadConfigBool(cmd *cobra.Command, flag string) (bool, error) {
	if DefineSource(cmd, flag) == ENV {
		boolVal, err := strconv.ParseBool(os.Getenv(GetFlagVarName(flag)))
		return boolVal, err
	}
	value, err := cmd.Flags().GetBool(flag)
	return value, err
}

// DefineSource - Calculate precedence rule for flags and variables
func DefineSource(cmd *cobra.Command, flag string) int {
	//Precedence rule:
	//1st Flag
	//2nd Variable
	//3rd default Variable value
	flagReference := cmd.Flags().Lookup(flag)
	if flagReference != nil && flagReference.Changed {
		log.Printf("Flag(%s) set from CLI flag", flag)
		return FLAG
	}

	envVarName := GetFlagVarName(flag)
	_, envExist := os.LookupEnv(envVarName)
	if envExist {
		log.Printf("Enviroment Variable(%s) set - using this value for Flag(%s)\n", envVarName, flag)
		return ENV
	}

	log.Printf("Enviroment Variable(%s) not set\n", flag)
	return NONE
}

// GetFlagVarName - Translates a flag name into a enviroment variable name
// based on ticket: https://github.com/kubefirst/kubefirst/issues/277
func GetFlagVarName(flag string) string {

	varName := "KUBEFIRST_" + strings.ToUpper(flag)
	varName = strings.ReplaceAll(varName, "-", "_")
	return varName
}
