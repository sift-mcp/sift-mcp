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
	result := s.pipeline.Process(ctx, report)

	if err := s.repository.Store(ctx, report); err != nil {
		return nil, err
	}

	if err := s.recordFingerprints(ctx, report, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) recordFingerprints(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) error {
	for _, failedTest := range result.FailedTests {
		if failedTest.HistoricalContext == nil || failedTest.HistoricalContext.ErrorFingerprint == "" {
			continue
		}
		err := s.repository.UpsertErrorFingerprint(ctx,
			failedTest.HistoricalContext.ErrorFingerprint,
			failedTest.NormalizedTrace,
			report.Timestamp,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ReAnalyze(ctx context.Context, report *core.TestReport) (*core.AnalysisResult, error) {
	result := s.pipeline.Process(ctx, report)
	return result, nil
}
