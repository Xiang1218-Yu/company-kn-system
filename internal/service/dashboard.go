package service

import (
	"context"

	"kn-system/internal/repository"
)

// DashboardService reports operational metrics for the ops panel. It is a thin
// read-through over the QARepo aggregation so the handler stays trivial.
type DashboardService struct {
	qa *repository.QARepo
}

func NewDashboardService(qa *repository.QARepo) *DashboardService {
	return &DashboardService{qa: qa}
}

func (s *DashboardService) Load(ctx context.Context) (repository.Dashboard, error) {
	return s.qa.LoadDashboard(ctx)
}
