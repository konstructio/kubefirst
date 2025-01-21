package generate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type Files struct {
	data map[string]bytes.Buffer
}

func (f *Files) Add(file string, content bytes.Buffer) {
	if f.data == nil {
		f.data = map[string]bytes.Buffer{}
	}

	f.data[file] = content
}

func (f *Files) Save(filePrefix string) error {
	for file, content := range f.data {
		name := filepath.Join(filePrefix, file)

		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(name, content.Bytes(), 0o644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}
