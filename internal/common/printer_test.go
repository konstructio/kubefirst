package common

import (
	"bytes"
	"io"
	"testing"
)

type FailingWriter struct{}

func (f *FailingWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (f *FailingWriter) String() string {
	return ""
}

func TestAwsCommand_Print(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		writers []io.Writer
		wantErr bool
	}{
		{
			name:    "write single string to single writer",
			input:   "test message",
			writers: []io.Writer{&bytes.Buffer{}},
			wantErr: false,
		},
		{
			name:    "write single string to multiple writers",
			input:   "test message",
			writers: []io.Writer{&bytes.Buffer{}, &bytes.Buffer{}},
			wantErr: false,
		},
		{
			name:    "write empty string",
			input:   "",
			writers: []io.Writer{&bytes.Buffer{}},
			wantErr: false,
		},
		{
			name:    "fail to write to writer",
			input:   "test message",
			writers: []io.Writer{&FailingWriter{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(tt.writers...)

			err := printer.Print(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WriteString() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				return
			}

			for i, w := range tt.writers {

				stringer := w.(interface {
					String() string
				})

				if got := stringer.String(); got != tt.input {
					t.Errorf("Writer %d got = %v, want %v", i, got, tt.input)
				}
			}
		})
	}
}
