package parser

import (
	"context"
	"io"
)

// PlainParser handles txt and markdown by reading the whole stream as UTF-8.
// Markdown is parsed as plain text: the chunker later splits on structure, and
// rendering/highlighting is a frontend concern, not a parsing one.
type PlainParser struct{}

func (PlainParser) Supports(fileType string) bool {
	switch fileType {
	case "txt", "md", "markdown", "text":
		return true
	default:
		return false
	}
}

func (PlainParser) Parse(ctx context.Context, r io.Reader) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
