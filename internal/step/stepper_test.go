package step

import (
	"bytes"
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
