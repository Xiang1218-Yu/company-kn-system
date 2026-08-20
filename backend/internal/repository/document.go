package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"kn-system/internal/model"
)

// DocumentRepo persists documents and their chunks. Vector search lives here so
// the service layer asks for "retrieve by vector" without knowing it is a
// cosine-distance SQL query.
type DocumentRepo struct {
	db *gorm.DB
}

func NewDocumentRepo(db *gorm.DB) *DocumentRepo { return &DocumentRepo{db: db} }

func (r *DocumentRepo) Create(ctx context.Context, d *model.Document) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DocumentRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	var d model.Document
	if err := r.db.WithContext(ctx).First(&d, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DocumentRepo) ListByKB(ctx context.Context, kbID uuid.UUID) ([]model.Document, error) {
	var docs []model.Document
	if err := r.db.WithContext(ctx).
		Where("kb_id = ?", kbID).
		Order("created_at desc").
		Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *DocumentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DocStatus, chunkCount int) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      status,
			"chunk_count": chunkCount,
		}).Error
}

func (r *DocumentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Document{}, "id = ?", id).Error
}

// Search runs a lightweight full-text search over document names within a
// knowledge base. It complements (not replaces) semantic search: keyword lookup
// is fast for known terms, while vectors cover meaning.
func (r *DocumentRepo) Search(ctx context.Context, kbID uuid.UUID, q string) ([]model.Document, error) {
	var docs []model.Document
	like := "%" + q + "%"
	if err := r.db.WithContext(ctx).
		Where("kb_id = ? AND name ILIKE ?", kbID, like).
		Order("created_at desc").
		Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// CreateChunks persists the chunks for a document in one transaction so a
// failure rolls back partial inserts — a document never ends up with a subset
// of its chunks indexed.
func (r *DocumentRepo) CreateChunks(ctx context.Context, chunks []model.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&chunks).Error
	})
}

// ReplaceChunks makes re-indexing idempotent: stale chunks are removed and the
// newly parsed set is inserted in the same transaction.
func (r *DocumentRepo) ReplaceChunks(ctx context.Context, docID uuid.UUID, chunks []model.Chunk) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("doc_id = ?", docID).Delete(&model.Chunk{}).Error; err != nil {
			return err
		}
		if len(chunks) == 0 {
			return nil
		}
		return tx.Create(&chunks).Error
	})
}

// DeleteChunks removes a document's chunks; called when re-indexing or on delete.
func (r *DocumentRepo) DeleteChunks(ctx context.Context, docID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("doc_id = ?", docID).Delete(&model.Chunk{}).Error
}

// RetrieveByVector runs cosine-distance similarity search against the chunks
// table, returning the top-k nearest neighbours for a question embedding. The
// <=> operator is pgvector's cosine distance; ordering ascending = most similar.
func (r *DocumentRepo) RetrieveByVector(ctx context.Context, kbID uuid.UUID, vec []float32, topK int) ([]model.Chunk, error) {
	var chunks []model.Chunk
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.* FROM chunks c
		JOIN documents d ON d.id = c.doc_id
		WHERE d.kb_id = ?
		ORDER BY c.vector <=> ?
		LIMIT ?`,
		kbID, pgvector.NewVector(vec), topK,
	).Scan(&chunks).Error
	return chunks, err
}
