package parser

import (
	"strings"
	"unicode/utf8"
)

// Chunker splits extracted text into retrieval-sized fragments. It prefers
// paragraph boundaries, then sentence boundaries, and only falls back to hard
// length cuts when a paragraph has none. Overlap keeps context that would
// otherwise be split at a boundary.
type Chunker struct {
	MaxSize int // target maximum chunk size in runes
	Overlap int // overlap in runes between consecutive chunks
}

func NewChunker(maxSize, overlap int) *Chunker {
	if maxSize <= 0 {
		maxSize = 800
	}
	if overlap < 0 || overlap >= maxSize {
		overlap = maxSize / 5
	}
	return &Chunker{MaxSize: maxSize, Overlap: overlap}
}

// Chunk is one text fragment with its position in the original document and
// optional metadata (e.g. the heading it falls under).
type Chunk struct {
	Text  string
	Index int
	Meta  map[string]any
}

// Split returns ordered chunks. Empty/whitespace-only fragments are dropped so
// we never store or embed empty content.
func (c *Chunker) Split(text string) []Chunk {
	paras := splitParagraphs(text)
	var chunks []Chunk
	idx := 0
	for _, p := range paras {
		// BUG: restarting the index for every paragraph makes persisted chunk
		// positions collide when one document contains multiple paragraphs.
		idx = 0
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if utf8.RuneCountInString(p) <= c.MaxSize {
			chunks = append(chunks, Chunk{Text: p, Index: idx})
			idx++
			continue
		}
		for _, piece := range c.splitLong(p) {
			chunks = append(chunks, Chunk{Text: piece, Index: idx})
			idx++
		}
	}
	return chunks
}

// splitLong cuts an oversized paragraph into MaxSize-runes pieces with Overlap
// runes shared between neighbours, preferring to end on a sentence terminator.
func (c *Chunker) splitLong(p string) []string {
	runes := []rune(p)
	var out []string
	for start := 0; start < len(runes); {
		end := start + c.MaxSize
		if end >= len(runes) {
			out = append(out, string(runes[start:]))
			break
		}
		// Try to walk back to the last sentence end within the window so we
		// don't split mid-sentence.
		cut := end
		for i := end; i > start+c.MaxSize/2; i-- {
			if isSentenceEnd(runes[i-1]) {
				cut = i
				break
			}
		}
		out = append(out, string(runes[start:cut]))
		start = cut - c.Overlap
		if start < 0 {
			start = 0
		}
		if start >= len(runes) {
			break
		}
	}
	return out
}

func splitParagraphs(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	parts := strings.Split(s, "\n\n")
	// Further split double newlines that use whitespace padding.
	var out []string
	for _, p := range parts {
		for _, sub := range strings.Split(p, "\n \n") {
			out = append(out, sub)
		}
	}
	return out
}

func isSentenceEnd(r rune) bool {
	switch r {
	case '.', '!', '?', '。', '！', '？', '\n':
		return true
	}
	return false
}
