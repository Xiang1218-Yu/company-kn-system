package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	apperr "kn-system/internal/errors"
	"kn-system/internal/model"
	"kn-system/internal/queue"
	"kn-system/internal/repository"
	"kn-system/internal/storage"
)

// DocumentService coordinates the upload, storage, listing, search, and
// deletion of documents. It deliberately does NOT do parsing/embedding itself —
// that is the IndexingService's job, triggered via the queue, so uploads return
// immediately and indexing happens asynchronously (a non-functional requirement).
type DocumentService struct {
	docs    *repository.DocumentRepo
	kbs     *repository.KBRepo
	storage storage.Storage
	queue   queue.Queue
	maxSize int64
}

func NewDocumentService(
	docs *repository.DocumentRepo,
	kbs *repository.KBRepo,
	store storage.Storage,
	q queue.Queue,
	maxSize int64,
) *DocumentService {
	return &DocumentService{
		docs:    docs,
		kbs:     kbs,
		storage: store,
		queue:   q,
		maxSize: maxSize,
	}
}

// SetQueue injects the queue after construction. This breaks a wiring cycle:
// the queue depends on the indexing service, and the document service depends
// on the queue; constructing the document service first with nil and injecting
// later lets both be created without a circular initializer.
func (s *DocumentService) SetQueue(q queue.Queue) {
	s.queue = q
}

// AllowedTypes is the set of upload formats the spec requires.
var AllowedTypes = map[string]bool{
	"pdf":  true,
	"docx": true,
	"md":   true,
	"txt":   true,
}

// Upload stores the file, creates the document record in pending status, and
// enqueues an indexing job. The multipart header carries size/type metadata so
// we don't have to read the whole body just to learn them.
func (s *DocumentService) Upload(ctx context.Context, kbID, userID uuid.UUID, fh *multipart.FileHeader) (*model.Document, error) {
	if _, err := s.kbs.FindByID(ctx, kbID); err != nil {
		return nil, apperr.New(apperr.KindNotFound, "knowledge base not found")
	}
	if fh.Size > s.maxSize {
		return nil, apperr.New(apperr.KindValidation, fmt.Sprintf("file exceeds %d bytes", s.maxSize))
	}
	fileType := normalizeType(fh.Filename)
	if !AllowedTypes[fileType] {
		return nil, apperr.New(apperr.KindValidation, "unsupported file type: "+fileType)
	}

	src, err := fh.Open()
	if err != nil {
		return nil, apperr.Wrap(apperr.KindInternal, "open upload", err)
	}
	defer src.Close()

	// Object key is <kbID>/<docID>/<original-name> so files are namespaced per
	// knowledge base and never collide across uploads of the same name.
	docID := uuid.New()
	key := fmt.Sprintf("%s/%s/%s", kbID, docID, filepath.Base(fh.Filename))
	if err := s.storage.Save(ctx, key, src, fh.Header.Get("Content-Type"), fh.Size); err != nil {
		return nil, apperr.Wrap(apperr.KindInternal, "store file", err)
	}

	doc := &model.Document{
		ID:         docID,
		KbID:       kbID,
		Name:       filepath.Base(fh.Filename),
		FilePath:   key,
		FileSize:   fh.Size,
		FileType:   fileType,
		Status:     model.DocStatusPending,
		UploadedBy: userID,
	}
	if err := s.docs.Create(ctx, doc); err != nil {
		// Best-effort cleanup of the stored object so we don't leak bytes when
		// the DB write fails; the user can retry the upload cleanly.
		_ = s.storage.Delete(ctx, key)
		return nil, apperr.Wrap(apperr.KindInternal, "create document", err)
	}

	if err := s.queue.Enqueue(ctx, queue.Job{DocumentID: docID, KbID: kbID}); err != nil {
		// Enqueue failure is recorded on the document so the user can see the
		// problem and retry, rather than leaving it forever "pending".
		_ = s.docs.UpdateStatus(ctx, docID, model.DocStatusFailed, 0)
		return nil, apperr.Wrap(apperr.KindInternal, "enqueue indexing", err)
	}
	return doc, nil
}

// normalizeType lowercases and strips the dot from the extension.
func normalizeType(name string) string {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	return strings.ToLower(ext)
}

func (s *DocumentService) List(ctx context.Context, kbID uuid.UUID) ([]model.Document, error) {
	return s.docs.ListByKB(ctx, kbID)
}

func (s *DocumentService) Get(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	d, err := s.docs.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindNotFound, "document not found", err)
	}
	return d, nil
}

func (s *DocumentService) Delete(ctx context.Context, id uuid.UUID) error {
	d, err := s.docs.FindByID(ctx, id)
	if err != nil {
		return apperr.Wrap(apperr.KindNotFound, "document not found", err)
	}
	// Order matters: delete chunks first (FK), then the file, then the record.
	// If the storage delete fails we still remove the DB record so the user is
	// not left with an orphan document pointing at a deleted file; the object
	// is best-effort reaped later.
	if err := s.docs.DeleteChunks(ctx, id); err != nil {
		return apperr.Wrap(apperr.KindInternal, "delete chunks", err)
	}
	_ = s.storage.Delete(ctx, d.FilePath)
	if err := s.docs.Delete(ctx, id); err != nil {
		return apperr.Wrap(apperr.KindInternal, "delete document", err)
	}
	return nil
}

// Download opens the stored object for streaming. The handler writes the bytes
// and sets content disposition; the service only resolves the stream.
func (s *DocumentService) Download(ctx context.Context, id uuid.UUID) (io.ReadCloser, *model.Document, error) {
	d, err := s.docs.FindByID(ctx, id)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.KindNotFound, "document not found", err)
	}
	rc, err := s.storage.Open(ctx, d.FilePath)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.KindInternal, "open file", err)
	}
	return rc, d, nil
}

// Search does keyword search over document names. It is distinct from semantic
// QA: here the user is browsing documents, not asking a question.
func (s *DocumentService) Search(ctx context.Context, kbID uuid.UUID, q string) ([]model.Document, error) {
	if strings.TrimSpace(q) == "" {
		return s.docs.ListByKB(ctx, kbID)
	}
	return s.docs.Search(ctx, kbID, q)
}

// RetryIndex re-enqueues a failed document for indexing, used by the "retry"
// affordance in the spec.
func (s *DocumentService) RetryIndex(ctx context.Context, id uuid.UUID) error {
	d, err := s.docs.FindByID(ctx, id)
	if err != nil {
		return apperr.Wrap(apperr.KindNotFound, "document not found", err)
	}
	if d.Status == model.DocStatusIndexing {
		return apperr.New(apperr.KindConflict, "document is already indexing")
	}
	if err := s.queue.Enqueue(ctx, queue.Job{DocumentID: d.ID, KbID: d.KbID}); err != nil {
		return apperr.Wrap(apperr.KindInternal, "enqueue indexing", err)
	}
	return s.docs.UpdateStatus(ctx, d.ID, model.DocStatusPending, d.ChunkCount)
}
