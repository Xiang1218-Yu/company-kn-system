package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// PDFParser extracts text from PDFs. The library reads in-memory, so the parser
// buffers the whole stream first. For very large PDFs this is a memory cost
// traded for simplicity; the chunker and queue run async so it does not block
// the upload request.
type PDFParser struct{}

func (PDFParser) Supports(fileType string) bool { return fileType == "pdf" }

func (PDFParser) Parse(ctx context.Context, r io.Reader) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read pdf: %w", err)
	}
	reader := bytes.NewReader(b)
	f, err := pdf.NewReader(reader, int64(len(b)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	var buf bytes.Buffer
	for i := 1; i <= f.NumPage(); i++ {
		page := f.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			// A single unreadable page should not abort the whole document.
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
	}
	return buf.String(), nil
}
