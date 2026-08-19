package service

import (
	"context"

	"github.com/google/uuid"

	apperr "kn-system/internal/errors"
	"kn-system/internal/model"
	"kn-system/internal/repository"
)

// KBService owns knowledge-base lifecycle rules: creation, update, deletion,
// and the membership/ownership checks that gate those operations.
type KBService struct {
	kbs *repository.KBRepo
}

func NewKBService(kbs *repository.KBRepo) *KBService { return &KBService{kbs: kbs} }

// CreateKBInput is the validated shape for creating a knowledge base.
type CreateKBInput struct {
	Name        string
	Description string
	TeamID      *uuid.UUID
	CreatedBy   uuid.UUID
}

func (s *KBService) Create(ctx context.Context, in CreateKBInput) (*model.KnowledgeBase, error) {
	if in.Name == "" {
		return nil, apperr.New(apperr.KindValidation, "name is required")
	}
	kb := &model.KnowledgeBase{
		Name:        in.Name,
		Description: in.Description,
		TeamID:      in.TeamID,
		CreatedBy:   in.CreatedBy,
	}
	if err := s.kbs.Create(ctx, kb); err != nil {
		return nil, apperr.Wrap(apperr.KindInternal, "create kb", err)
	}
	return kb, nil
}

func (s *KBService) List(ctx context.Context) ([]model.KnowledgeBase, error) {
	return s.kbs.List(ctx)
}

func (s *KBService) Get(ctx context.Context, id uuid.UUID) (*model.KnowledgeBase, error) {
	kb, err := s.kbs.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindNotFound, "knowledge base not found", err)
	}
	return kb, nil
}

// UpdateKBInput carries the editable fields of a knowledge base.
type UpdateKBInput struct {
	Name        string
	Description string
}

func (s *KBService) Update(ctx context.Context, id uuid.UUID, in UpdateKBInput) (*model.KnowledgeBase, error) {
	kb, err := s.kbs.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindNotFound, "knowledge base not found", err)
	}
	if in.Name != "" {
		kb.Name = in.Name
	}
	if in.Description != "" {
		kb.Description = in.Description
	}
	if err := s.kbs.Update(ctx, kb); err != nil {
		return nil, apperr.Wrap(apperr.KindInternal, "update kb", err)
	}
	return kb, nil
}

// Delete removes a knowledge base and (via FK cascade) its documents and chunks.
// It is restricted to admins/owners at the handler layer.
func (s *KBService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.kbs.Delete(ctx, id); err != nil {
		return apperr.Wrap(apperr.KindInternal, "delete kb", err)
	}
	return nil
}
