/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// String in slice returns true if the string is in the slice.
func StringInSlice(s string, slice []string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}

	return false
}

func CreateK1ClusterDirectory(clusterName string) {
	// Create k1 dir if it doesn't exist
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
	}
	k1Dir := fmt.Sprintf("%s/.k1/%s", homePath, clusterName)
	if _, err := os.Stat(k1Dir); os.IsNotExist(err) {
		err := os.MkdirAll(k1Dir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", k1Dir)
		}
	}
}

// CreateDirIfNotExists creates a directory if it doesn't exists already.
func CreateDirIfNotExists(d string) {
	if _, err := os.Stat(d); os.IsNotExist(err) {
		err := os.MkdirAll(d, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", d)
		}
	}
}

// EnvOrDefault returns the value of the specified env var or the default value.
func EnvOrDefault(env, def string) string {
	v, ok := os.LookupEnv(env)
	if !ok {
		return def
	}

	return v
}

func ParseJSONToMap(jsonStr string) (map[string][]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	for key, value := range result {
		switch v := value.(type) {
		case map[string]interface{}, []interface{}: // For nested structures, marshal back to JSON
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			secretData[key] = bytes
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			secretData[key] = bytes
		}
	}

	return secretData, nil
}
