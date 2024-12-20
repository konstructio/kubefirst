package common

import (
	"fmt"
	"io"
)

type Printer struct {
	writers []io.Writer
}

func NewPrinter(writers ...io.Writer) *Printer {
	return &Printer{
		writers: writers,
	}
}

func (p *Printer) AddWriter(w io.Writer) {
	p.writers = append(p.writers, w)
}

func (p *Printer) Print(s string) error {
	for _, w := range p.writers {
		_, err := fmt.Fprint(w, s)
		if err != nil {
			return fmt.Errorf("failed to write to writer: %w", err)
		}
	}

	return nil
}
