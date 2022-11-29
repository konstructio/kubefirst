package pkg

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
)

// ZerologSetup setup Zerolog and return the configured Zerolog instance
func ZerologSetup(logFile *os.File) zerolog.Logger {
	// short file path/name
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

	return log.Output(zerolog.ConsoleWriter{Out: logFile, NoColor: false}).With().Caller().Logger()
}
