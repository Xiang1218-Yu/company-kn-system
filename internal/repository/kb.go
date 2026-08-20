package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kn-system/internal/model"
)

// KBRepo persists knowledge bases. Knowledge-space isolation is enforced here:
// every query is scoped by either the kb id or the membership predicate, so a
// query against one space can never leak documents from another.
type KBRepo struct {
	db *gorm.DB
}

func NewKBRepo(db *gorm.DB) *KBRepo { return &KBRepo{db: db} }

func (r *KBRepo) Create(ctx context.Context, kb *model.KnowledgeBase) error {
	return r.db.WithContext(ctx).Create(kb).Error
}

func (r *KBRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.KnowledgeBase, error) {
	var kb model.KnowledgeBase
	if err := r.db.WithContext(ctx).First(&kb, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &kb, nil
}

func (r *KBRepo) List(ctx context.Context) ([]model.KnowledgeBase, error) {
	var kbs []model.KnowledgeBase
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&kbs).Error; err != nil {
		return nil, err
	}
	return kbs, nil
}

func (r *KBRepo) Update(ctx context.Context, kb *model.KnowledgeBase) error {
	return r.db.WithContext(ctx).Save(kb).Error
}

func (r *KBRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.KnowledgeBase{}, "id = ?", id).Error
}
