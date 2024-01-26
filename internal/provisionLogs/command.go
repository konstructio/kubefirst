/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provisionLogs

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

type Log struct {
	Level   string `bson:"level" json:"level"`
	Time    string `bson:"time" json:"time"`
	Message string `bson:"message" json:"message"`
}

var (
	color      = termenv.EnvColorProfile().Color
	infoStyle  = termenv.Style{}.Foreground(color("27")).Styled
	errorStyle = termenv.Style{}.Foreground(color("196")).Styled
	timeStyle  = termenv.Style{}.Foreground(color("245")).Bold().Styled
	textStyle  = termenv.Style{}.Foreground(color("15")).Styled
)

func AddLog(logMsg string) {
	log := Log{}
	formatterMsg := ""

	err := json.Unmarshal([]byte(logMsg), &log)
	if err != nil {
		formatterMsg = textStyle(logMsg)
	} else {
		parsedTime, err := time.Parse(time.RFC3339, log.Time)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			return
		}

		// Format the parsed time into the desired format
		formattedDateStr := parsedTime.Format("2006-01-02 15:04:05")

		timeLog := timeStyle(formattedDateStr)
		level := infoStyle(strings.ToUpper(log.Level))

		if log.Level == "error" {
			level = errorStyle(strings.ToUpper(log.Level))
		}

		message := textStyle(log.Message)

		formatterMsg = fmt.Sprintf("%s %s: %s", timeLog, level, message)
	}

	renderedMessage := formatterMsg

	ProvisionLogs.Send(logMessage{message: renderedMessage})
}
