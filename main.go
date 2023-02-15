/*
Copyright © 2022 Kubefirst Inc. devops@kubefirst.com
*/
package main

import (
	"fmt"
	stdLog "log"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/cmd"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func main() {
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
	logsFolder := fmt.Sprintf("%s/logs", k1Dir)
	// we're ignoring folder creation handling at the moment
	// todo: add folder creation handler
	_ = os.Mkdir(logsFolder, 0700)

	logfile := fmt.Sprintf("%s/log_%d.log", logsFolder, epoch)
	//fmt.Printf("Logging at: %s \n", logfile)

	// Avoid printing log helper for certain subcommands
	var excludeLogHelperFrom []string = []string{"version"}
	if len(os.Args) > 1 && !pkg.FindStringInSlice(excludeLogHelperFrom, os.Args[1]) {
		fmt.Printf("\n-----------\n")
		fmt.Printf("Follow your logs with: \n   tail -f  %s \n", logfile)
		fmt.Printf("\n-----------\n")
	}

	file, err := pkg.OpenLogFile(logfile)
	if err != nil {
		stdLog.Panicf("unable to store log location, error is: %s", err)
	}

	// handle file close request
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Print(err)
		}
	}(file)

	// setup default logging
	// this Go standard log is active to keep compatibility with current code base
	stdLog.SetOutput(file)
	stdLog.SetPrefix("LOG: ")
	stdLog.SetFlags(stdLog.Ldate | stdLog.Lmicroseconds | stdLog.Llongfile)

	// setup Zerolog
	log.Logger = pkg.ZerologSetup(file, zerolog.InfoLevel)

	config := configs.ReadConfig()
	// setup Viper (for non-local resources)
	if err = pkg.SetupViper(config); err != nil {
		stdLog.Panic(err)
	}

	viper.Set("logs-location", logsFolder)
	err = viper.WriteConfig()
	if err != nil {
		stdLog.Panicf("unable to set log-file-location, error is: %s", err)
	}

	cmd.Execute()
}
