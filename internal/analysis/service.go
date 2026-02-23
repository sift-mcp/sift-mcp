package analysis

import (
	"context"

	"sift/internal/core"
)

type Service struct {
	pipeline   *Pipeline
	repository core.ReportRepository
}

func NewService(pipeline *Pipeline, repository core.ReportRepository) *Service {
	return &Service{
		pipeline:   pipeline,
		repository: repository,
	}
}

func (s *Service) Analyze(ctx context.Context, report *core.TestReport) (*core.AnalysisResult, error) {
	if err := s.repository.Store(ctx, report); err != nil {
		return nil, err
	}

	result := s.pipeline.Process(ctx, report)
	return result, nil
}

func (s *Service) ReAnalyze(ctx context.Context, report *core.TestReport) (*core.AnalysisResult, error) {
	result := s.pipeline.Process(ctx, report)
	return result, nil
}
