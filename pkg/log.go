package pkg

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ZerologSetup setup Zerolog and return the configured Zerolog instance
func ZerologSetup(logFile *os.File) zerolog.Logger {
	// short file path/name
	// it seems longer name doesn't work as expected.
	/*
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
			return file + ":" + strconv.Itoa(line)
		}
	*/
	// default log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	return log.Output(zerolog.ConsoleWriter{Out: logFile, NoColor: false, TimeFormat: "2006-01-02T15:04"}).With().Timestamp().Caller().Logger()
	//return log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false}).With().Caller().Logger()
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
