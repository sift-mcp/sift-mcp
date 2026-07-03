package stages

import (
	"context"
	"time"

	"sift/internal/core"
)

type EnrichHistoryStage struct {
	repo core.ReportRepository
}

func NewEnrichHistoryStage(repo core.ReportRepository) *EnrichHistoryStage {
	return &EnrichHistoryStage{repo: repo}
}

func (s *EnrichHistoryStage) Name() string {
	return "enrich_history"
}

func (s *EnrichHistoryStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	if s.repo == nil {
		return
	}

	asOf := report.Timestamp
	oneDayAgo := asOf.Add(-24 * time.Hour)
	sevenDaysAgo := asOf.Add(-7 * 24 * time.Hour)

	totalReports, _ := s.repo.Count(ctx)

	for i := range result.FailedTests {
		failedTest := &result.FailedTests[i]

		if failedTest.HistoricalContext == nil {
			failedTest.HistoricalContext = &core.HistoricalContext{}
		}

		count24h, err := s.repo.GetTestFailureCounts(ctx, failedTest.Name, oneDayAgo, report.ID)
		if err == nil {
			failedTest.HistoricalContext.FailureCount24h = count24h
		}

		count7d, err := s.repo.GetTestFailureCounts(ctx, failedTest.Name, sevenDaysAgo, report.ID)
		if err == nil {
			failedTest.HistoricalContext.FailureCount7d = count7d
		}

		lastSuccess, err := s.repo.GetTestLastSuccess(ctx, failedTest.Name)
		if err == nil && !lastSuccess.IsZero() {
			failedTest.HistoricalContext.LastSeen = lastSuccess
		}

		fingerprint := failedTest.HistoricalContext.ErrorFingerprint
		if fingerprint != "" {
			occurrences, firstSeen, lastSeen, err := s.repo.GetErrorFingerprint(ctx, fingerprint)
			if err == nil && occurrences > 0 {
				failedTest.HistoricalContext.FirstSeen = firstSeen
				failedTest.HistoricalContext.LastSeen = lastSeen
			}
		}

		if totalReports > 0 {
			failureReports, _ := s.repo.GetFailureHistory(ctx, failedTest.Name, int(totalReports))
			passCount := int(totalReports) - len(failureReports)
			failedTest.HistoricalContext.IsFlaky = len(failureReports) > 0 && passCount > 0
		}
	}
}
