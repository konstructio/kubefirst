/*
Copyright © 2022 Kubefirst Inc. devops@kubefirst.com
*/
package main

import (
	"fmt"
	"github.com/kubefirst/kubefirst/cmd/cli"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/spf13/cobra"
	"log"
	"os"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func main() {
	now := time.Now()
	epoch := now.Unix()

	currentFolder, err := os.Getwd()
	if err != nil {
		log.Panicf("unable to get current folder location, error is: %s", err)
	}
	logsFolder := fmt.Sprintf("%s/%s", currentFolder, "logs")
	// we're ignoring folder creation handling at the moment
	// todo: add folder creation handler
	_ = os.Mkdir(logsFolder, 0700)

	logfile := fmt.Sprintf("%s/log_%d.log", logsFolder, epoch)
	fmt.Printf("Logging at: %s \n", logfile)

	config := configs.ReadConfig()

	err = pkg.SetupViper(config)
	if err != nil {
		log.Panic(err)
	}

	viper.Set("log-folder-location", logsFolder)
	err = viper.WriteConfig()
	if err != nil {
		log.Panicf("unable to set log-file-location, error is: %s", err)
	}

	file, err := openLogFile(logfile)
	if err != nil {
		log.Panicf("unable to store log location, error is: %s", err)
	}

	// handle file close request
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	}(file)

	// setup logging
	log.SetOutput(file)
	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	// progress bar
	progressPrinter.GetInstance()

	// todo: what is this for?
	cobra.OnInitialize()

	coreKubefirstCmd := cli.NewCommand()
	if err = coreKubefirstCmd.Execute(); err != nil {
		os.Exit(1)
	}

	//toolsKubefirstCmd := cli.NewToolCommand()
	//if err = toolsKubefirstCmd.Execute(); err != nil {
	//	os.Exit(1)
	//}
	//cmd.Execute()
}

func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}
