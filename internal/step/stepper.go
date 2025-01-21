package step

import (
	"io"

	"github.com/konstructio/cli-utils/stepper"
)

type Stepper interface {
	NewStep(stepName string) *stepper.Step
}

type StepFactory struct {
	writer io.Writer
}

func NewStepFactory(writer io.Writer) *StepFactory {
	return &StepFactory{writer: writer}
}

func (sf *StepFactory) NewStep(stepName string) *stepper.Step {
	return stepper.New(sf.writer, stepName)
}
