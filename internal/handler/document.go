package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kn-system/internal/middleware"
	"kn-system/internal/service"
	"kn-system/pkg/response"
)

// DocumentHandler exposes document upload, listing, search, download, and
// delete. The max upload size is configurable so this handler does not hardcode
// a limit; it receives the limit from the server wiring.
type DocumentHandler struct {
	svc *service.DocumentService
}

func NewDocumentHandler(svc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// Upload godoc
// POST /api/v1/kb/:id/documents
//
// Reads the multipart form, delegates storage + enqueue to the service, and
// returns the freshly-created (pending) document so the frontend can poll its
// index status. The :id path param is the knowledge-base id (named id, not
// kbId, to share the param position with other kb routes).
func (h *DocumentHandler) Upload(c *gin.Context) {
	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid kb id")
		return
	}
	u, _ := middleware.CurrentUser(c)

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file field 'file' is required")
		return
	}
	doc, err := h.svc.Upload(c.Request.Context(), kbID, u.ID, file)
	if err != nil {
		emit(c, err)
		return
	}
	response.Created(c, doc)
}

// List godoc
// GET /api/v1/kb/:id/documents
func (h *DocumentHandler) List(c *gin.Context) {
	kbID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid kb id")
		return
	}
	docs, err := h.svc.List(c.Request.Context(), kbID)
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, docs)
}

// Get godoc
// GET /api/v1/documents/:id
func (h *DocumentHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	doc, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, doc)
}

// Status godoc
// GET /api/v1/documents/:id/status
//
// A dedicated status endpoint (rather than embedding in Get) keeps the common
// polling path cheap: the frontend hits this every second without pulling the
// full record.
func (h *DocumentHandler) Status(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	doc, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, gin.H{
		"status":      doc.Status,
		"chunk_count": doc.ChunkCount,
	})
}

// Delete godoc
// DELETE /api/v1/documents/:id
func (h *DocumentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		emit(c, err)
		return
	}
	response.NoContent(c)
}

// Download godoc
// GET /api/v1/documents/:id/download
//
// Streams the stored object back to the client with a content-disposition header
// set to the original filename so browsers save it under the right name. We copy
// the bytes directly (not via c.Stream) so the connection closes cleanly once the
// file is fully written; c.Stream's continue-flag semantics confused some
// clients into waiting on a never-closing chunked body.
func (h *DocumentHandler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	rc, doc, err := h.svc.Download(c.Request.Context(), id)
	if err != nil {
		emit(c, err)
		return
	}
	defer rc.Close()
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, doc.Name))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", doc.FileSize))
	c.Status(http.StatusOK)
	// Copy directly into the response writer; once EOF is reached the handler
	// returns and Gin flushes and closes the connection.
	if _, err := io.Copy(c.Writer, rc); err != nil {
		// Header already sent; we can only log.
		c.Error(err)
		return
	}
}

// Search godoc
// GET /api/v1/search?q=&kbId=
func (h *DocumentHandler) Search(c *gin.Context) {
	kbID, err := uuid.Parse(c.Query("kbId"))
	if err != nil {
		response.BadRequest(c, "kbId query param is required")
		return
	}
	q := c.Query("q")
	docs, err := h.svc.Search(c.Request.Context(), kbID, q)
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, docs)
}

// Retry godoc
// POST /api/v1/documents/:id/reindex
//
// Re-enqueues a failed document. Added beyond the spec's listed endpoints
// because the spec explicitly calls for retry on index failure.
func (h *DocumentHandler) Retry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.RetryIndex(c.Request.Context(), id); err != nil {
		emit(c, err)
		return
	}
	response.OK(c, gin.H{"status": "requeued"})
}
