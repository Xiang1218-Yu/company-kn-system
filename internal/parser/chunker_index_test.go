package parser

import (
	"strings"
	"testing"
)

func TestChunkerKeepsGlobalChunkIndexesForMultiParagraphDocument(t *testing.T) {
	chunker := NewChunker(32, 4)
	chunks := chunker.Split(strings.Repeat("第一段内容。 ", 5) + "\n\n第二段内容。")
	if len(chunks) < 2 {
		t.Fatalf("Split() returned %d chunks, want multiple paragraphs", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Fatalf("chunk %d has index %d, want global index %d", i, chunk.Index, i)
		}
	}
}
