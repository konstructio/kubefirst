package flagset

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const CONFIG = 2
const ENV = 1
const FLAG = 0
const NONE = 99

// ReadConfigString - Read a Cobra-CLI flag or env variable of type string, precedence rule (Flags first, envVar second)
// default value is ""
// Current version doesn't have error, but we may expect some as the function evolves.
func ReadConfigString(cmd *cobra.Command, flag string) (string, error) {
	source := DefineSource(cmd, flag)
	if source == CONFIG {
		return viper.GetString(GetConfig(flag)), nil
	}
	if source == ENV {
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
	source := DefineSource(cmd, flag)
	if source == CONFIG {
		return viper.GetBool(GetConfig(flag)), nil
	}
	if source == ENV {
		boolVal, err := strconv.ParseBool(os.Getenv(GetFlagVarName(flag)))
		return boolVal, err
	}
	value, err := cmd.Flags().GetBool(flag)
	return value, err
}

// DefineSource - Calculate precedence rule for flags and variables
func DefineSource(cmd *cobra.Command, flag string) int {
	//Precedence rule:
	//1st Config file
	//2nd Flag
	//3rd Variable
	//4th default Variable value
	configReference := viper.Get(GetConfig(flag))
	if configReference != nil {
		log.Printf("Flag(%s) set from Config File", flag)
		return CONFIG
	}

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

func GetConfig(flag string) string {
	return "config." + flag
}

// InjectConfigs - Append configs from a new source to the main viper file.
func InjectConfigs(extraConfig string) {
	//Workaround due to: https://github.com/spf13/viper/issues/181
	var v = viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(extraConfig)
	_ = v.ReadInConfig()
	//log.Println(v.AllSettings())
	viper.MergeConfigMap(v.AllSettings())
}
