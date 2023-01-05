/*
Copyright © 2022 Kubefirst Inc. devops@kubefirst.com
*/
package main

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/reports"
	stdLog "log"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/cmd"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func main() {
	now := time.Now()
	epoch := now.Unix()

	currentFolder, err := os.Getwd()
	if err != nil {
		stdLog.Panicf("unable to get current folder location, error is: %s", err)
	}
	logsFolder := fmt.Sprintf("%s/%s", currentFolder, "logs")
	// we're ignoring folder creation handling at the moment
	// todo: add folder creation handler
	_ = os.Mkdir(logsFolder, 0700)

	logfile := fmt.Sprintf("%s/log_%d.log", logsFolder, epoch)

	msg := fmt.Sprintf("Follow your logs with: tail -f %s", logfile)
	fmt.Println(reports.StyleMessage(msg))

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
	log.Logger = pkg.ZerologSetup(file)

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
