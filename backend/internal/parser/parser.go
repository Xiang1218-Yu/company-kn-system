package parser

import (
	"context"
	"io"
	"strings"
)

// Parser extracts plain text from a document stream. Implementations must
// close any internal resources they open. The returned text is the concatenation
// of every readable paragraph; structure (headings, tables) is flattened unless
// a parser is sophisticated enough to preserve it.
type Parser interface {
	// Supports reports whether this parser handles the given file type
	// (extension without the dot, e.g. "pdf", "md").
	Supports(fileType string) bool
	// Parse reads from r and returns the document text. The caller owns the
	// returned slice; parsers must not retain the reader.
	Parse(ctx context.Context, r io.Reader) (string, error)
}

// Registry selects the right parser by file type. Keeping the dispatcher here
// means the service layer stays unaware of how many parsers exist.
type Registry struct {
	parsers []Parser
}

func NewRegistry(parsers ...Parser) *Registry {
	return &Registry{parsers: parsers}
}

// For returns the parser for a file type, or an error if none matches. The
// default markdown/text fallback is registered last so unknown-but-plain
// extensions are not silently dropped.
func (r *Registry) For(fileType string) (Parser, error) {
	ft := strings.ToLower(strings.TrimPrefix(fileType, "."))
	for _, p := range r.parsers {
		if p.Supports(ft) {
			return p, nil
		}
	}
	return nil, &UnsupportedError{FileType: fileType}
}

// UnsupportedError signals an unsupported file type so the caller can map it to
// a validation error rather than a generic internal failure.
type UnsupportedError struct{ FileType string }

func (e *UnsupportedError) Error() string {
	return "unsupported file type: " + e.FileType
}
