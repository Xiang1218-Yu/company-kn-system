package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/pgvector/pgvector-go"
	"kn-system/internal/embedder"
	"kn-system/internal/model"
	"kn-system/internal/parser"
	"kn-system/internal/queue"
	"kn-system/internal/repository"
	"kn-system/internal/storage"

	"go.uber.org/zap"

	"kn-system/pkg/logger"
)

// IndexingService implements queue.Processor. It owns the document→chunks
// pipeline: read bytes from storage, parse to text, chunk, embed, persist. It
// is the only place that composes parser + embedder + chunk repo, so changing
// the retrieval strategy is a change here, not across the app.
type IndexingService struct {
	docs     *repository.DocumentRepo
	store    storage.Storage
	registry *parser.Registry
	chunker  *parser.Chunker
	emb      embedder.Embedder
}

func NewIndexingService(
	docs *repository.DocumentRepo,
	store storage.Storage,
	registry *parser.Registry,
	chunker *parser.Chunker,
	emb embedder.Embedder,
) *IndexingService {
	return &IndexingService{
		docs:     docs,
		store:    store,
		registry: registry,
		chunker:  chunker,
		emb:      emb,
	}
}

// Process runs the indexing pipeline for one document. It marks the document
// indexing→indexed/failed so the frontend can show live progress. Failures
// here are retried by the queue wrapper before being recorded as failed.
func (s *IndexingService) Process(ctx context.Context, job queue.Job) error {
	doc, err := s.docs.FindByID(ctx, job.DocumentID)
	if err != nil {
		return fmt.Errorf("load document: %w", err)
	}
	if err := s.docs.UpdateStatus(ctx, doc.ID, model.DocStatusIndexing, doc.ChunkCount); err != nil {
		return fmt.Errorf("mark indexing: %w", err)
	}

	rc, err := s.store.Open(ctx, doc.FilePath)
	if err != nil {
		s.fail(ctx, doc.ID)
		return fmt.Errorf("open file: %w", err)
	}
	defer rc.Close()

	p, err := s.registry.For(doc.FileType)
	if err != nil {
		s.fail(ctx, doc.ID)
		return err
	}
	text, err := p.Parse(ctx, rc)
	if err != nil {
		s.fail(ctx, doc.ID)
		return fmt.Errorf("parse: %w", err)
	}

	pieces := s.chunker.Split(text)
	if len(pieces) == 0 {
		// An empty/whitespace document produces no chunks; mark indexed with
		// zero so the user sees a clean success rather than a hanging pending.
		return s.docs.UpdateStatus(ctx, doc.ID, model.DocStatusIndexed, 0)
	}

	chunks := make([]model.Chunk, 0, len(pieces))
	for _, pc := range pieces {
		vec, err := s.emb.Embed(ctx, pc.Text)
		if err != nil {
			s.fail(ctx, doc.ID)
			return fmt.Errorf("embed chunk %d: %w", pc.Index, err)
		}
		chunks = append(chunks, model.Chunk{
			DocID:      doc.ID,
			Content:    pc.Text,
			ChunkIndex: pc.Index,
			Metadata:   pc.Meta,
			Vector:     pgvector.NewVector(vec),
		})
	}
	if err := s.docs.CreateChunks(ctx, chunks); err != nil {
		s.fail(ctx, doc.ID)
		return fmt.Errorf("persist chunks: %w", err)
	}
	if err := s.docs.UpdateStatus(ctx, doc.ID, model.DocStatusIndexed, len(chunks)); err != nil {
		return fmt.Errorf("mark indexed: %w", err)
	}
	logger.L.Info("document indexed",
		zap.Stringer("doc", doc.ID),
		zap.Int("chunks", len(chunks)))
	return nil
}

// fail marks the document failed so the UI shows the error and offers retry.
func (s *IndexingService) fail(ctx context.Context, id uuid.UUID) {
	_ = s.docs.UpdateStatus(ctx, id, model.DocStatusFailed, 0)
}
