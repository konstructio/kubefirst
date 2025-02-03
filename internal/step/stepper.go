package step

import (
	"fmt"
	"io"

	"github.com/konstructio/cli-utils/stepper"
)

const (
	EmojiCheck   = "✅"
	EmojiError   = "🔴"
	EmojiMagic   = "✨"
	EmojiHead    = "🤕"
	EmojiNoEntry = "⛔"
	EmojiTada    = "🎉"
	EmojiAlarm   = "⏰"
	EmojiBug     = "🐛"
	EmojiBulb    = "💡"
	EmojiWarning = "⚠️"
	EmojiWrench  = "🔧"
	EmojiBook    = "📘"
)

type Stepper interface {
	NewProgressStep(stepName string)
	FailCurrentStep(err error)
	CompleteCurrentStep()
	InfoStep(emoji, message string)
	InfoStepString(message string)
	DisplayLogHints(cloudProvider string, estimatedTime int)
}

type Factory struct {
	writer      io.Writer
	currentStep *stepper.Step
}

func NewStepFactory(writer io.Writer) *Factory {
	return &Factory{writer: writer}
}

func (s *Factory) NewProgressStep(stepName string) {
	if s.currentStep == nil {
		s.currentStep = stepper.New(s.writer, stepName)
	} else if s.currentStep != nil && s.currentStep.GetName() != stepName {
		s.currentStep.Complete(nil)
		s.currentStep = stepper.New(s.writer, stepName)
	}
}

func (s *Factory) FailCurrentStep(err error) {
	s.currentStep.Complete(err)
}

func (s *Factory) CompleteCurrentStep() {
	s.currentStep.Complete(nil)
}

func (s *Factory) GetCurrentStep() string {
	return s.currentStep.GetName()
}

func (s *Factory) InfoStep(emoji, message string) {
	fmt.Fprintf(s.writer, "%s %s\n", emoji, message)
}

func (s *Factory) InfoStepString(message string) {
	fmt.Fprintf(s.writer, "%s\n", message)
}

func (s *Factory) DisplayLogHints(cloudProvider string, estimatedTime int) {
	documentationLink := "https://kubefirst.konstruct.io/docs/"

	if cloudProvider != "" {
		documentationLink += cloudProvider + `/quick-start/install/cli`
	}

	header := "\n Welcome to Kubefirst \n\n"

	verboseLogs := fmt.Sprintf("%s To view verbose logs run below command in new terminal: \"kubefirst logs\"\n%s Documentation: %s\n\n", EmojiBulb, EmojiBook, documentationLink)

	estimatedTimeMsg := fmt.Sprintf("%s Estimated time: %d minutes\n\n", EmojiAlarm, estimatedTime)

	s.InfoStepString(fmt.Sprintf("%s%s%s", header, verboseLogs, estimatedTimeMsg))
}
