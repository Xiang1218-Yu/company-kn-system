package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

// DocxParser extracts text from Word .docx files. A .docx is a ZIP archive; the
// document body lives at word/document.xml and paragraph runs are wrapped in
// <w:t> elements. This lightweight extractor avoids a heavy dependency while
// covering the common case of text-only documents.
type DocxParser struct{}

func (DocxParser) Supports(fileType string) bool { return fileType == "docx" }

func (DocxParser) Parse(ctx context.Context, r io.Reader) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read docx: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}
	var body []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open document.xml: %w", err)
			}
			body, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("read document.xml: %w", err)
			}
			break
		}
	}
	if body == nil {
		return "", fmt.Errorf("docx: word/document.xml not found")
	}
	return extractText(string(body)), nil
}

// extractText pulls the contents of every <w:t> element and inserts paragraph
// breaks at </w:p> so chunks split on real paragraph boundaries.
func extractText(xml string) string {
	var b strings.Builder
	i := 0
	for i < len(xml) {
		lt := strings.Index(xml[i:], "<")
		if lt < 0 {
			break
		}
		i += lt
		gt := strings.Index(xml[i:], ">")
		if gt < 0 {
			break
		}
		tag := xml[i : i+gt+1]
		i += gt + 1
		if strings.HasPrefix(tag, "<w:t") {
			end := strings.Index(xml[i:], "<")
			if end >= 0 {
				b.WriteString(unescape(xml[i : i+end]))
				i += end
			}
		} else if strings.HasPrefix(tag, "</w:p>") {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}
