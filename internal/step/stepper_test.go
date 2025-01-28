package step

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepFactory_NewStep(t *testing.T) {
	tests := []struct {
		name     string
		stepName string
	}{
		{
			name:     "creates new step with name",
			stepName: "test step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			sf := &StepFactory{writer: buf}

			step := sf.NewProgressStep(tt.stepName)

			assert.NotNil(t, step)
			assert.Equal(t, tt.stepName, step.GetName())
		})
	}
}

func TestStepFactory_DisplayLogHints(t *testing.T) {
	type fields struct {
		writer io.Writer
	}
	type args struct {
		logFile       string
		cloudProvider string
		estimatedTime int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "displays log hints",
			fields: fields{
				writer: os.Stderr,
			},
			args: args{
				logFile:       "test.log",
				cloudProvider: "test",
				estimatedTime: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := &StepFactory{
				writer: tt.fields.writer,
			}
			sf.DisplayLogHints(tt.args.logFile, tt.args.cloudProvider, tt.args.estimatedTime)
		})
	}
}
