package parser

import (
	"context"
	"errors"
	"io"
	"testing"
)

type readerThatMustNotBeRead struct {
	reads int
}

func (r *readerThatMustNotBeRead) Read(_ []byte) (int, error) {
	r.reads++
	return 0, errors.New("reader was consumed after cancellation")
}

func TestParsersStopBeforeReadingAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parsers := []struct {
		name   string
		parser Parser
	}{
		{name: "plain", parser: PlainParser{}},
		{name: "docx", parser: DocxParser{}},
		{name: "pdf", parser: PDFParser{}},
	}
	for _, tc := range parsers {
		t.Run(tc.name, func(t *testing.T) {
			reader := &readerThatMustNotBeRead{}
			_, err := tc.parser.Parse(ctx, reader)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Parse() error = %v, want context.Canceled", err)
			}
			if reader.reads != 0 {
				t.Fatalf("Parse() read %d time(s) after cancellation", reader.reads)
			}
		})
	}
}

var _ io.Reader = (*readerThatMustNotBeRead)(nil)
