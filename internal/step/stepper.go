package step

import (
	"fmt"
	"io"
	"strings"

	"github.com/konstructio/cli-utils/stepper"
)

const (
	emojiCheck   = "✅"
	emojiError   = "🔴"
	emojiMagic   = "✨"
	emojiHead    = "🤕"
	emojiNoEntry = "⛔"
	emojiTada    = "🎉"
	emojiAlarm   = "⏰"
	emojiBug     = "🐛"
	emojiBulb    = "💡"
	emojiWarning = "⚠️"
	emojiWrench  = "🔧"
	emojiBook    = "📘"
)

type Stepper interface {
	NewProgressStep(stepName string) *stepper.Step
	InfoStep(emoji, message string)
}

type StepFactory struct {
	writer io.Writer
}

func NewStepFactory(writer io.Writer) *StepFactory {
	return &StepFactory{writer: writer}
}

func (sf *StepFactory) NewProgressStep(stepName string) *stepper.Step {
	return stepper.New(sf.writer, stepName)
}

func (sf *StepFactory) InfoStep(emoji, message string) {
	fmt.Fprintf(sf.writer, "%s %s\n", emoji, message)
}

func (sf *StepFactory) InfoStepString(message string) {
	fmt.Fprintf(sf.writer, "%s\n", message)
}

func (sf *StepFactory) DisplayLogHints(logFile, cloudProvider string, estimatedTime int) {
	documentationLink := "https://kubefirst.konstruct.io/docs/"

	if cloudProvider != "" {
		documentationLink += cloudProvider + `/quick-start/install/cli`
	}

	header := `
##
# Welcome to Kubefirst
`

	verboseLogs := fmt.Sprintf("### %s To view verbose logs run below command in new terminal: \"kubefirst logs\"\n%s Documentation: %s\n\n", emojiBulb, emojiBook, documentationLink)

	estimatedTimeMsg := fmt.Sprintf("### %s Estimated time: %d minutes\n\n", emojiAlarm, estimatedTime)

	sf.InfoStepString(strings.Join([]string{header, verboseLogs, estimatedTimeMsg}, ""))

}
