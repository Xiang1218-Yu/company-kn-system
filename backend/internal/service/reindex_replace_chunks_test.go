package service

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kn-system/internal/model"
	"kn-system/internal/parser"
	"kn-system/internal/queue"
	"kn-system/internal/repository"
	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

type repeatIndexStorage struct{ contents string }

func (s repeatIndexStorage) Save(context.Context, string, io.Reader, string, int64) error { return nil }
func (s repeatIndexStorage) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString(s.contents)), nil
}
func (s repeatIndexStorage) Delete(context.Context, string) error { return nil }

type repeatIndexEmbedder struct{}

func (repeatIndexEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0.25}, nil
}
func (repeatIndexEmbedder) Dim() int { return 1 }

func TestReindexReplacesExistingDocumentChunks(t *testing.T) {
	logger.L = zap.NewNop()
	db, err := gorm.Open(sqlite.Open("file:reindex_replace_chunks?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.Document{}, &model.Chunk{}); err != nil {
		t.Fatalf("migrate document tables: %v", err)
	}
	docs := repository.NewDocumentRepo(db)
	doc := model.Document{
		ID: uuid.New(), KbID: uuid.New(), UploadedBy: uuid.New(),
		Name: "handbook.txt", FilePath: "uploads/handbook.txt", FileSize: 8,
		FileType: "txt", Status: model.DocStatusPending,
	}
	if err := docs.Create(context.Background(), &doc); err != nil {
		t.Fatalf("create document: %v", err)
	}

	svc := NewIndexingService(
		docs,
		repeatIndexStorage{contents: "one durable chunk"},
		parser.NewRegistry(parser.PlainParser{}),
		parser.NewChunker(100, 0),
		repeatIndexEmbedder{},
	)
	job := queue.Job{DocumentID: doc.ID, KbID: doc.KbID}
	// IndexingService.Process is the async indexing worker entry point.
	if err := svc.Process(context.Background(), job); err != nil {
		t.Fatalf("first index pass: %v", err)
	}
	if err := svc.Process(context.Background(), job); err != nil {
		t.Fatalf("second index pass: %v", err)
	}

	var count int64
	if err := db.Model(&model.Chunk{}).Where("doc_id = ?", doc.ID).Count(&count).Error; err != nil {
		t.Fatalf("count reindexed chunks: %v", err)
	}
	if count != 1 {
		t.Fatalf("reindex retained duplicate chunks: got %d, want 1", count)
	}
}
