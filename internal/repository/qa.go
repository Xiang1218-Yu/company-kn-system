package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kn-system/internal/model"
)

// QARepo persists question/answer logs. It also reports the operations metrics
// (counts, satisfaction) used by the dashboard endpoint.
type QARepo struct {
	db *gorm.DB
}

func NewQARepo(db *gorm.DB) *QARepo { return &QARepo{db: db} }

func (r *QARepo) Create(ctx context.Context, log *model.QALog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *QARepo) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.QALog, error) {
	var logs []model.QALog
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *QARepo) SetFeedback(ctx context.Context, id uuid.UUID, fb model.Feedback) error {
	return r.db.WithContext(ctx).Model(&model.QALog{}).
		Where("id = ?", id).
		Update("feedback", fb).Error
}

// Dashboard is the aggregated metric set for the operations panel.
type Dashboard struct {
	DocumentCount int64   `json:"document_count"`
	QACount       int64   `json:"qa_count"`
	Satisfaction  float64 `json:"satisfaction"`
}

// LoadDashboard computes counts and satisfaction ratio in a single call. The
// satisfaction ratio is upvotes over voted answers; unvoted answers are
// excluded rather than counted as negative.
func (r *QARepo) LoadDashboard(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	if err := r.db.WithContext(ctx).Model(&model.Document{}).Count(&d.DocumentCount).Error; err != nil {
		return d, err
	}
	if err := r.db.WithContext(ctx).Model(&model.QALog{}).Count(&d.QACount).Error; err != nil {
		return d, err
	}
	var up, total int64
	row := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FILTER (WHERE feedback = 'up'), COUNT(*) FILTER (WHERE feedback IN ('up','down')) FROM qa_logs`).Row()
	if err := row.Scan(&up, &total); err != nil {
		return d, err
	}
	if total > 0 {
		d.Satisfaction = float64(up) / float64(total)
	}
	return d, nil
}
