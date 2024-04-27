package logging

import (
	"fmt"
	stdLog "log"
	"os"
	"time"

	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	zeroLog "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Init sets up log directory and logfile.
func Init() {
	// user has not specified any action, no need to setup a logfile
	if len(os.Args[1:]) == 0 {
		return
	}

	config := configs.ReadConfig()

	if err := pkg.SetupViper(config, true); err != nil {
		stdLog.Panic(err)
	}

	now := time.Now()
	epoch := now.Unix()
	logfileName := fmt.Sprintf("log_%d.log", epoch)
	isProvision := utilities.StringInSlice("create", os.Args)
	isLogs := utilities.StringInSlice("logs", os.Args)

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
		log.Info().Msg(err.Error())
	}

	//* create k1Dir if it doesn't exist
	k1Dir := fmt.Sprintf("%s/.k1", homePath)
	if _, err := os.Stat(k1Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(k1Dir, os.ModePerm); err != nil {
			log.Info().Msgf("%s directory already exists, continuing", k1Dir)
		}
	}

	//* create log directory
	logsFolder := fmt.Sprintf("%s/logs", k1Dir)
	utilities.CreateDirIfNotExists(logsFolder)

	//* create session log file
	logfile := fmt.Sprintf("%s/%s", logsFolder, logfileName)
	logFileObj, err := pkg.OpenLogFile(logfile)
	if err != nil {
		stdLog.Panicf("unable to store log location, error is: %s - please verify the current user has write access to this directory", err)
	}

	// // handle file close request
	// defer func(logFileObj *os.File) {
	// 	if err = logFileObj.Close(); err != nil {
	// 		log.Print(err)
	// 	}
	// }(logFileObj)

	// setup default logging
	// this Go standard log is active to keep compatibility with current code base
	stdLog.SetOutput(logFileObj)
	stdLog.SetPrefix("LOG: ")
	stdLog.SetFlags(stdLog.Ldate)

	log.Logger = zeroLog.New(logFileObj).With().Timestamp().Logger()

	viper.Set("k1-paths.logs-dir", logsFolder)
	viper.Set("k1-paths.log-file", logfile)
	viper.Set("k1-paths.log-file-name", logfileName)

	if viper.WriteConfig() != nil {
		stdLog.Panicf("unable to set log-file-location, error is: %s", err)
	}
}
