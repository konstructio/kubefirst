/*
Copyright © 2022 Kubefirst Inc. devops@kubefirst.com

*/
package main

import (
	"errors"
	"fmt"
	"github.com/kubefirst/kubefirst/cmd"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"time"
)

func main() {
	now := time.Now()
	epoch := now.Unix()

	currentFolder, err := os.Getwd()
	if err != nil {
		log.Panic().Msgf("unable to get current folder location, error is: %s", err)
	}
	logsFolder := fmt.Sprintf("%s/%s", currentFolder, "logs")
	// we're ignoring folder creation handling at the moment
	// todo: add folder creation handler
	_ = os.Mkdir(logsFolder, 0700)

	logfile := fmt.Sprintf("%s/log_%d.log", logsFolder, epoch)
	fmt.Printf("Logging at: %s \n", logfile)

	file, err := openLogFile(logfile)
	if err != nil {
		//log.Panicf("unable to store log location, error is: %s", err)
	}

	// handle file close request
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Err(err).Send()
		}
	}(file)

	// setup logging with color and code line on logs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: file}).With().Caller().Logger()

	config := configs.ReadConfig()
	err = pkg.SetupViper(config)
	if err != nil {
		log.Panic().Err(err).Send()
	}

	viper.Set("log-folder-location", logsFolder)
	err = viper.WriteConfig()
	if err != nil {
		log.Panic().Msgf("unable to set log-file-location, error is: %s", err)
	}

	log.Info().Msgf("info example")
	log.Err(errors.New("error msg")).Send()
	log.Warn().Msg("warning")
	log.Debug().Str("Service Context", "ArgoCD").Msg("this is happening here")

	cmd.Execute()
}

func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}
