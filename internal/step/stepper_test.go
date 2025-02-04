package step

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStepFactory_NewStep(t *testing.T) {
	t.Run("creates new step with name", func(t *testing.T) {
		stepName := "test step"
		buf := &bytes.Buffer{}
		sf := NewStepFactory(buf)

		sf.NewProgressStep(stepName)

		assert.Equal(t, stepName, sf.GetCurrentStep())
	})

	t.Run("should create new step if provided step name", func(t *testing.T) {
		stepName := "test step"
		buf := &bytes.Buffer{}
		sf := NewStepFactory(buf)

		sf.NewProgressStep(stepName)

		assert.Equal(t, stepName, sf.GetCurrentStep())

		newStepName := "another step"
		sf.NewProgressStep(newStepName)

		assert.Equal(t, newStepName, sf.GetCurrentStep())

		assert.Eventually(t, func() bool { return assert.Contains(t, buf.String(), stepName) }, 1*time.Second, 100*time.Millisecond)
		assert.Eventually(t, func() bool { return assert.Contains(t, buf.String(), newStepName) }, 1*time.Second, 100*time.Millisecond)
	})

	t.Run("should not change current step if provided same name", func(t *testing.T) {
		stepName := "test step"
		buf := &bytes.Buffer{}
		sf := NewStepFactory(buf)

		sf.NewProgressStep(stepName)

		assert.Equal(t, stepName, sf.GetCurrentStep())

		sf.NewProgressStep(stepName)

		assert.Equal(t, stepName, sf.GetCurrentStep())
	})
}

func TestStepFactory_FailCurrentStep(t *testing.T) {
	t.Run("should fail current step", func(t *testing.T) {
		stepName := "test step"
		errorMessage := "test error"
		buf := &bytes.Buffer{}
		sf := NewStepFactory(buf)

		sf.NewProgressStep(stepName)

		sf.FailCurrentStep(fmt.Errorf("%s", errorMessage))

		assert.Contains(t, buf.String(), errorMessage)
	})
}

func TestStepFactory_CompleteCurrentStep(t *testing.T) {
	t.Run("should complete current step", func(t *testing.T) {
		stepName := "test step"
		buf := &bytes.Buffer{}
		sf := NewStepFactory(buf)

		sf.NewProgressStep(stepName)

		sf.CompleteCurrentStep()

		assert.Eventually(t, func() bool { return assert.Contains(t, buf.String(), EmojiCheck) }, 1*time.Second, 100*time.Millisecond)
	})
}

func TestStepFactory_DisplayLogHints(t *testing.T) {
	type fields struct {
		writer io.Writer
	}
	type args struct {
		cloudProvider string
		estimatedTime int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "displays log hints without blowing up",
			fields: fields{
				writer: os.Stderr,
			},
			args: args{
				cloudProvider: "test",
				estimatedTime: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := &Factory{
				writer: tt.fields.writer,
			}
			sf.DisplayLogHints(tt.args.cloudProvider, tt.args.estimatedTime)

			// no assertions, just make sure it doesn't blow up
		})
	}
}
