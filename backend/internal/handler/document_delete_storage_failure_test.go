package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kn-system/internal/model"
	"kn-system/internal/repository"
	"kn-system/internal/service"
)

type deleteFailingStorage struct {
	deleteCalls int
}

func (s *deleteFailingStorage) Save(context.Context, string, io.Reader, string, int64) error {
	return nil
}
func (s *deleteFailingStorage) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}
func (s *deleteFailingStorage) Delete(context.Context, string) error {
	s.deleteCalls++
	return errors.New("object storage unavailable")
}

func TestDeleteDocumentKeepsMetadataWhenObjectRemovalFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:delete_storage_failure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.Document{}, &model.Chunk{}); err != nil {
		t.Fatalf("migrate document tables: %v", err)
	}

	docs := repository.NewDocumentRepo(db)
	doc := model.Document{
		ID: uuid.New(), KbID: uuid.New(), UploadedBy: uuid.New(),
		Name: "policy.txt", FilePath: "uploads/policy.txt", FileSize: 12,
		FileType: "txt", Status: model.DocStatusIndexed, ChunkCount: 1,
	}
	if err := docs.Create(context.Background(), &doc); err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := docs.CreateChunks(context.Background(), []model.Chunk{{DocID: doc.ID, Content: "policy", ChunkIndex: 0}}); err != nil {
		t.Fatalf("create document chunk: %v", err)
	}

	store := &deleteFailingStorage{}
	h := NewDocumentHandler(service.NewDocumentService(docs, nil, store, nil, 0))
	router := gin.New()
	// DELETE /api/v1/documents/:id
	router.DELETE("/api/v1/documents/:id", h.Delete)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/documents/"+doc.ID.String(), nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("delete endpoint returned %d after storage deletion failure, want %d", rec.Code, http.StatusInternalServerError)
	}
	if store.deleteCalls != 1 {
		t.Errorf("storage delete calls = %d, want 1", store.deleteCalls)
	}
	if _, err := docs.FindByID(context.Background(), doc.ID); err != nil {
		t.Errorf("document metadata was removed after storage deletion failure: %v", err)
	}
	var chunks int64
	if err := db.Model(&model.Chunk{}).Where("doc_id = ?", doc.ID).Count(&chunks).Error; err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if chunks != 1 {
		t.Errorf("document chunks were removed after storage deletion failure: got %d, want 1", chunks)
	}
}
