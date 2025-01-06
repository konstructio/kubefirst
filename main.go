/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"
	stdLog "log"
	"os"
	"time"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/cmd"
	zeroLog "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

func main() {
	if err := run(); err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}
}

func run() error {
	argsWithProg := os.Args

	config, err := configs.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := utils.SetupViper(config, true); err != nil {
		return fmt.Errorf("failed to setup Viper: %w", err)
	}

	now := time.Now()
	epoch := now.Unix()
	logfileName := fmt.Sprintf("log_%d.log", epoch)

	isProvision := slices.Contains(argsWithProg, "create")
	isLogs := slices.Contains(argsWithProg, "logs")

	// don't create a new log file for logs, using the previous one
	if isLogs {
		logfileName = viper.GetString("k1-paths.log-file-name")
	}

	// use cluster name as filename
	if isProvision {
		clusterName := fmt.Sprint(epoch)
		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]

			// Check if the argument is "--cluster-name"
			if arg == "--cluster-name" && i+1 < len(os.Args) {
				// Get the value of the cluster name
				clusterName = os.Args[i+1]
				break
			}
		}

		logfileName = fmt.Sprintf("log_%s.log", clusterName)
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	k1Dir := fmt.Sprintf("%s/.k1", homePath)

	// * create k1Dir if it doesn't exist
	if _, err := os.Stat(k1Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(k1Dir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %q: %w", k1Dir, err)
		}
	}

	// * create log directory if it doesn't exist
	logsFolder := fmt.Sprintf("%s/logs", k1Dir)
	if _, err := os.Stat(logsFolder); os.IsNotExist(err) {
		if err := os.Mkdir(logsFolder, 0o700); err != nil {
			return fmt.Errorf("error creating logs directory: %w", err)
		}
	}

	// * create session log file
	logfile := fmt.Sprintf("%s/%s", logsFolder, logfileName)
	logFileObj, err := utils.OpenLogFile(logfile)
	if err != nil {
		return fmt.Errorf("unable to store log location, error is: %w - please verify the current user has write access to this directory", err)
	}

	// handle file close request
	defer func(logFileObj *os.File) {
		if err := logFileObj.Close(); err != nil {
			log.Error().Msgf("error closing log file: %v", err)
		}
	}(logFileObj)

	// setup default logging
	// this Go standard log is active to keep compatibility with current code base
	stdLog.SetOutput(logFileObj)
	stdLog.SetPrefix("LOG: ")
	stdLog.SetFlags(stdLog.Ldate)

	log.Logger = zeroLog.New(logFileObj).With().Timestamp().Logger()

	viper.Set("k1-paths.logs-dir", logsFolder)
	viper.Set("k1-paths.log-file", logfile)
	viper.Set("k1-paths.log-file-name", logfileName)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return cmd.Execute()
}
