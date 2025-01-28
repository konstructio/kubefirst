package step

import (
	"fmt"
	"io"

	"github.com/konstructio/cli-utils/stepper"
)

const (
	EMOJI_CHECK    = "✅"
	EMOJI_ERROR    = "🔴"
	EMOJI_MAGIC    = "✨"
	EMOJI_HEAD     = "🤕"
	EMOJI_NO_ENTRY = "⛔"
	EMOJI_TADA     = "🎉"
	EMOJI_ALARM    = "⏰"
	EMOJI_BUG      = "🐛"
	EMOJI_BULB     = "💡"
	EMOJI_WARNING  = "⚠️"
	EMOJI_WRENCH   = "🔧"
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
