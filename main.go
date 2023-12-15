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

	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/cmd"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/spf13/viper"
)

func main() {
	argsWithProg := os.Args

	bubbleTeaBlacklist := []string{"completion", "help", "--help", "-h", "quota"}
	canRunBubbleTea := true

	if argsWithProg != nil {
		for _, arg := range argsWithProg {
			isBlackListed := slices.Contains(bubbleTeaBlacklist, arg)

			if isBlackListed {
				canRunBubbleTea = false
			}
		}
	}

	now := time.Now()
	epoch := now.Unix()

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
	}

	k1Dir := fmt.Sprintf("%s/.k1", homePath)

	//* create k1Dir if it doesn't exist
	if _, err := os.Stat(k1Dir); os.IsNotExist(err) {
		err := os.MkdirAll(k1Dir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", k1Dir)
		}
	}

	//* create log directory
	logsFolder := fmt.Sprintf("%s/logs", k1Dir)
	_ = os.Mkdir(logsFolder, 0700)
	if err != nil {
		log.Fatal().Msgf("error creating logs directory: %s", err)
	}

	//* create session log file
	logfile := fmt.Sprintf("%s/log_%d.log", logsFolder, epoch)
	logFileObj, err := pkg.OpenLogFile(logfile)
	if err != nil {
		stdLog.Panicf("unable to store log location, error is: %s - please verify the current user has write access to this directory", err)
	}

	// handle file close request
	defer func(logFileObj *os.File) {
		err = logFileObj.Close()
		if err != nil {
			log.Print(err)
		}
	}(logFileObj)

	// setup default logging
	// this Go standard log is active to keep compatibility with current code base
	stdLog.SetOutput(logFileObj)
	stdLog.SetPrefix("LOG: ")
	stdLog.SetFlags(stdLog.Ldate | stdLog.Lmicroseconds | stdLog.Llongfile)

	// setup Zerolog
	log.Logger = pkg.ZerologSetup(logFileObj, zerolog.InfoLevel)

	config := configs.ReadConfig()
	if err = pkg.SetupViper(config); err != nil {
		stdLog.Panic(err)
	}

	viper.Set("k1-paths.logs-dir", logsFolder)
	viper.Set("k1-paths.log-file", fmt.Sprintf("%s/log_%d.log", logsFolder, epoch))
	err = viper.WriteConfig()
	if err != nil {
		stdLog.Panicf("unable to set log-file-location, error is: %s", err)
	}

	if canRunBubbleTea {
		progress.InitializeProgressTerminal()

		go func() {
			cmd.Execute()
		}()

		progress.Progress.Run()
	} else {
		cmd.Execute()
	}

}
