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

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k8s"
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

	// cmd.Execute()
	kcfg := k8s.CreateKubeConfig(false, "/Users/scott/.kube/config")
	yamlData, err := kcfg.KustomizeBuild("/Users/scott/src/scratch/k-apply-clgo")
	if err != nil {
		fmt.Printf("error yamldata: %s", err)
	}

	output, err := kcfg.SplitYAMLFile(yamlData)
	if err != nil {
		fmt.Println(err)
	}

	err = kcfg.ApplyObjects("", output)
	if err != nil {
		fmt.Printf("error apply: %s", err)
	}
}
