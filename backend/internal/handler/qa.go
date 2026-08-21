package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kn-system/internal/middleware"
	"kn-system/internal/model"
	"kn-system/internal/service"
	"kn-system/pkg/response"
)

// QAHandler exposes the question endpoints: streaming ask, history, feedback.
type QAHandler struct {
	svc *service.QAService
}

func NewQAHandler(svc *service.QAService) *QAHandler {
	return &QAHandler{svc: svc}
}

// Ask godoc
// POST /api/v1/qa/ask  (SSE: text/event-stream)
//
// The endpoint streams answer tokens as SSE `delta` events, then a final
// `done` event carrying the sources and the qa log id. The frontend can
// render tokens as they arrive and attach feedback once done.
//
// If the Accept header does not request event-stream we fall back to a single
// JSON response so the endpoint is usable from curl and non-SSE clients.
func (h *QAHandler) Ask(c *gin.Context) {
	var req struct {
		KbID     string `json:"kb_id" binding:"required"`
		Question string `json:"question" binding:"required"`
		History  []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		} `json:"history"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "kb_id and question are required")
		return
	}
	kbID, err := uuid.Parse(req.KbID)
	if err != nil {
		response.BadRequest(c, "invalid kb_id")
		return
	}
	u, _ := middleware.CurrentUser(c)

	history := make([][2]string, 0, len(req.History))
	for _, t := range req.History {
		history = append(history, [2]string{t.Question, t.Answer})
	}

	if c.GetHeader("Accept") == "text/event-stream" {
		h.streamAnswer(c, service.AskInput{
			UserID: u.ID, KbID: kbID, Question: req.Question, History: history,
		})
		return
	}

	// Non-streaming fallback: one shot.
	res, err := h.svc.Ask(context.Background(), service.AskInput{
		UserID: u.ID, KbID: kbID, Question: req.Question, History: history,
	})
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, res)
}

// streamAnswer writes SSE events. Each token is a `delta` event; the final
// `done` event carries sources + log id so the client can show citations and
// wire up feedback. The function owns the response writer until done.
func (h *QAHandler) streamAnswer(c *gin.Context, in service.AskInput) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable proxy buffering
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// No streaming support: degrade to the non-streaming path.
		res, err := h.svc.Ask(context.Background(), in)
		if err != nil {
			emit(c, err)
			return
		}
		writeSSE(c.Writer, "delta", res.Answer)
		writeSSE(c.Writer, "done", res)
		return
	}

	res, err := h.svc.StreamAnswer(context.Background(), in, func(tok string) {
		writeSSE(c.Writer, "delta", tok)
		if flusher != nil {
			flusher.Flush()
		}
	})
	if err != nil {
		// Surface the error as a terminal SSE event so the client can render it.
		writeSSE(c.Writer, "error", gin.H{"message": "stream failed"})
		return
	}
	writeSSE(c.Writer, "done", res)
	flusher.Flush()
}

// writeSSE serializes one server-sent event. Data is JSON-encoded so the client
// parses a consistent shape regardless of event type.
func writeSSE(w io.Writer, event string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
}

// History godoc
// GET /api/v1/qa/history
func (h *QAHandler) History(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	logs, err := h.svc.History(c.Request.Context(), u.ID, 50)
	if err != nil {
		emit(c, err)
		return
	}
	response.OK(c, logs)
}

// Feedback godoc
// POST /api/v1/qa/feedback
func (h *QAHandler) Feedback(c *gin.Context) {
	var req struct {
		LogID    string `json:"log_id" binding:"required"`
		Feedback string `json:"feedback" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "log_id and feedback are required")
		return
	}
	logID, err := uuid.Parse(req.LogID)
	if err != nil {
		response.BadRequest(c, "invalid log_id")
		return
	}
	if err := h.svc.SetFeedback(c.Request.Context(), logID, model.Feedback(req.Feedback)); err != nil {
		emit(c, err)
		return
	}
	response.NoContent(c)
}
