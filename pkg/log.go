/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ZerologSetup setup Zerolog and return the configured Zerolog instance
func ZerologSetup(logFile *os.File, level zerolog.Level) zerolog.Logger {
	zerolog.CallerMarshalFunc = shortCallerMarshalFunc
	zerolog.SetGlobalLevel(level)
	return log.Output(zerolog.ConsoleWriter{Out: logFile, NoColor: true, TimeFormat: "2006-01-02T15:04"}).With().Timestamp().Caller().Logger()
}

// shortCallerMarshalFunc is a custom marshal function for zerolog.CallerMarshalFunc variable.
// It formats the file path and line number of the log caller in a shortened form.
// It takes in three parameters:
//
//	pc uintptr representing the program counter
//	file string representing the file path of the caller
//	line int representing the line number of the caller
//
// It returns a string in the format of shortPackageName/file.extension:line
func shortCallerMarshalFunc(pc uintptr, file string, line int) string {
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	funcName := runtime.FuncForPC(pc).Name()
	packageName := funcName[:strings.LastIndex(funcName, ".")]
	stringCount := strings.Count(packageName, ".")
	if stringCount > 1 {
		packageName = strings.Replace(packageName, ".", "/", stringCount)
	}
	splitPath := strings.Split(packageName, "/")
	shortPackageName := strings.Join(splitPath[3:], "/")
	return shortPackageName + "/" + file + ":" + strconv.Itoa(line)
}

// GetLogLevelByString receives a log level string, and returns it's related zerolog Level iota
// panic (zerolog.PanicLevel, 5)
// fatal (zerolog.FatalLevel, 4)
// error (zerolog.ErrorLevel, 3)
// warn (zerolog.WarnLevel, 2)
// info (zerolog.InfoLevel, 1)
// debug (zerolog.DebugLevel, 0)
// trace (zerolog.TraceLevel, -1)
func GetLogLevelByString(logLevel string) zerolog.Level {

	level := make(map[string]zerolog.Level)
	level["trace"] = zerolog.TraceLevel
	level["debug"] = zerolog.DebugLevel
	level["info"] = zerolog.InfoLevel
	level["warning"] = zerolog.WarnLevel
	level["error"] = zerolog.ErrorLevel
	level["fatal"] = zerolog.FatalLevel
	level["panic"] = zerolog.PanicLevel

	return level[logLevel]

}
