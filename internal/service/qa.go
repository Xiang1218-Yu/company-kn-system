package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kn-system/internal/embedder"
	apperr "kn-system/internal/errors"
	"kn-system/internal/llm"
	"kn-system/internal/model"
	"kn-system/internal/repository"
	"kn-system/pkg/logger"
)

// QAService answers questions: embed the question, retrieve top-k chunks,
// assemble a prompt with citations, and stream the LLM's answer back. It also
// persists the Q&A log for history and dashboard metrics.
type QAService struct {
	docs *repository.DocumentRepo
	qa   *repository.QARepo
	emb  embedder.Embedder
	llm  llm.LLM
	topK int
}

func NewQAService(
	docs *repository.DocumentRepo,
	qa *repository.QARepo,
	emb embedder.Embedder,
	llm llm.LLM,
	topK int,
) *QAService {
	if topK <= 0 {
		topK = 5
	}
	return &QAService{docs: docs, qa: qa, emb: emb, llm: llm, topK: topK}
}

// AskInput is everything needed to answer a question.
type AskInput struct {
	UserID   uuid.UUID
	KbID     uuid.UUID
	Question string
	// History is optional prior turns for follow-up questions. Each pair is
	// user→assistant in order.
	History [][2]string
}

// AskResult carries the full answer (for the non-streaming fallback) and the
// sources used, plus the qa log id for later feedback.
type AskResult struct {
	LogID   uuid.UUID
	Answer  string
	Sources []model.SourceRef
}

// RetrieveAndBuildPrompt does the retrieval half: embed question, fetch chunks,
// format the grounding prompt. Factored out so streaming and non-streaming
// callers share one retrieval path.
func (s *QAService) RetrieveAndBuildPrompt(ctx context.Context, in AskInput) (string, []model.SourceRef, []llm.Message, error) {
	if strings.TrimSpace(in.Question) == "" {
		return "", nil, nil, apperr.New(apperr.KindValidation, "question is empty")
	}
	qVec, err := s.emb.Embed(ctx, in.Question)
	if err != nil {
		return "", nil, nil, apperr.Wrap(apperr.KindInternal, "embed question", err)
	}
	chunks, err := s.docs.RetrieveByVector(ctx, in.KbID, qVec, s.topK)
	if err != nil {
		return "", nil, nil, apperr.Wrap(apperr.KindInternal, "retrieve chunks", err)
	}

	sources, contextBlock := s.assembleSources(ctx, chunks)
	messages := s.buildMessages(in.Question, contextBlock, in.History)
	return contextBlock, sources, messages, nil
}

// assembleSources formats retrieved chunks into a citation list and a text
// block for the prompt. Documents are looked up so we can show the document
// name (not just the id) in citations.
func (s *QAService) assembleSources(ctx context.Context, chunks []model.Chunk) ([]model.SourceRef, string) {
	if len(chunks) == 0 {
		return nil, ""
	}
	// Cache document lookups so we don't query the same doc twice when chunks
	// come from the same document (the common case).
	cache := make(map[uuid.UUID]string)
	var sources []model.SourceRef
	var b strings.Builder
	for i, ch := range chunks {
		name, ok := cache[ch.DocID]
		if !ok {
			if d, err := s.docs.FindByID(ctx, ch.DocID); err == nil {
				name = d.Name
			} else {
				name = ch.DocID.String()
			}
			cache[ch.DocID] = name
		}
		snippet := snippet(ch.Content, 200)
		sources = append(sources, model.SourceRef{
			DocumentID:   ch.DocID,
			DocumentName: name,
			Snippet:      snippet,
			ChunkIndex:   ch.ChunkIndex,
		})
		fmt.Fprintf(&b, "[%d] (%s)\n%s\n\n", i+1, name, ch.Content)
	}
	return sources, b.String()
}

// buildMessages composes the system+user messages sent to the LLM. The system
// message instructs the model to ground answers in the provided context and to
// say when it cannot — matching the spec's "no answer" requirement.
func (s *QAService) buildMessages(question, contextBlock string, history [][2]string) []llm.Message {
	sys := "你是一个企业知识库助手。请根据下方提供的知识库片段回答用户问题。" +
		"如果片段中没有相关信息，请直接说明“知识库中暂无相关信息”，不要编造内容。" +
		"回答时可引用片段编号。"
	var msgs []llm.Message
	msgs = append(msgs, llm.Message{Role: "system", Content: sys})
	if contextBlock != "" {
		msgs = append(msgs, llm.Message{
			Role:    "system",
			Content: "以下是检索到的知识库片段：\n\n" + contextBlock,
		})
	}
	// Replay history for follow-up continuity. Iterate oldest→newest so
	// the model reads each prior question before the answer it prompted,
	// preserving the causal order of the conversation up to the current turn.
	for _, h := range history {
		msgs = append(msgs,
			llm.Message{Role: "user", Content: h[0]},
			llm.Message{Role: "assistant", Content: h[1]},
		)
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: question})
	return msgs
}

// Ask is the non-streaming answer path. Used as a fallback when a client cannot
// consume SSE, and as the canonical "produce + persist" routine.
func (s *QAService) Ask(ctx context.Context, in AskInput) (AskResult, error) {
	start := time.Now()
	_, sources, messages, err := s.RetrieveAndBuildPrompt(ctx, in)
	if err != nil {
		return AskResult{}, err
	}
	answer, err := s.llm.Complete(ctx, messages)
	if err != nil {
		// Degrade gracefully: never leak provider errors; surface a friendly
		// retry message per the availability non-functional requirement.
		answer = "服务繁忙，请稍后重试。"
		logger.L.Error("llm complete failed, returning degraded answer", zap.Error(err))
	}
	log := &model.QALog{
		UserID:       in.UserID,
		KbID:         in.KbID,
		Question:     in.Question,
		Answer:       answer,
		Sources:      sources,
		ResponseTime: int(time.Since(start).Milliseconds()),
	}
	if err := s.qa.Create(ctx, log); err != nil {
		// Logging failure must not fail the answer the user already received.
		logger.L.Error("persist qa log failed", zap.Error(err))
	}
	return AskResult{LogID: log.ID, Answer: answer, Sources: sources}, nil
}

// StreamAnswer retrieves context and streams tokens to onToken, then persists
// the assembled answer. The log is written once the stream completes so we
// capture the full text rather than a truncated buffer.
func (s *QAService) StreamAnswer(ctx context.Context, in AskInput, onToken func(string)) (AskResult, error) {
	start := time.Now()
	_, sources, messages, err := s.RetrieveAndBuildPrompt(ctx, in)
	if err != nil {
		return AskResult{}, err
	}
	var b strings.Builder
	streamErr := s.llm.Stream(ctx, messages, func(tok string) {
		b.WriteString(tok)
		onToken(tok)
	})
	answer := b.String()
	if streamErr != nil {
		// If the stream failed partway, still record what we have and append
		// the degradation note so the saved log is honest.
		if answer == "" {
			answer = "服务繁忙，请稍后重试。"
		}
		logger.L.Error("llm stream failed", zap.Error(streamErr))
	}
	log := &model.QALog{
		UserID:       in.UserID,
		KbID:         in.KbID,
		Question:     in.Question,
		Answer:       answer,
		Sources:      sources,
		ResponseTime: int(time.Since(start).Milliseconds()),
	}
	if err := s.qa.Create(ctx, log); err != nil {
		logger.L.Error("persist qa log failed", zap.Error(err))
	}
	return AskResult{LogID: log.ID, Answer: answer, Sources: sources}, nil
}

// History returns a user's recent questions.
func (s *QAService) History(ctx context.Context, userID uuid.UUID, limit int) ([]model.QALog, error) {
	return s.qa.ListByUser(ctx, userID, limit)
}

// SetFeedback records a user's up/down vote on an answer.
func (s *QAService) SetFeedback(ctx context.Context, logID uuid.UUID, fb model.Feedback) error {
	switch fb {
	case model.FeedbackUp, model.FeedbackDown, model.FeedbackNone:
	default:
		return apperr.New(apperr.KindValidation, "invalid feedback")
	}
	return s.qa.SetFeedback(ctx, logID, fb)
}

// snippet trims a chunk to a preview length for citation display.
func snippet(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}
