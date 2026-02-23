package stages

import (
	"context"
	"strings"
	"time"

	"sift/internal/core"
)

type ExtractFailuresStage struct{}

func NewExtractFailuresStage() *ExtractFailuresStage {
	return &ExtractFailuresStage{}
}

func (s *ExtractFailuresStage) Name() string {
	return "extract_failures"
}

func (s *ExtractFailuresStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	result.TotalTests = report.TotalTests
	result.Failed = report.Failed + report.Errored
	result.Passed = report.Passed
	result.Skipped = report.Skipped

	for _, suite := range report.Suites {
		for _, tc := range suite.Cases {
			if tc.Status != core.TestStatusFailed && tc.Status != core.TestStatusErrored {
				continue
			}

			errorClassification := ""
			errorSummary := ""

			if tc.Failure != nil {
				errorClassification = extractErrorType(tc.Failure)
				errorSummary = smartTruncate(tc.Failure.Message, 200)
			}

			info := core.FailedTestInfo{
				Name:                tc.Name,
				Classname:           tc.Classname,
				ErrorClassification: errorClassification,
				ErrorSummary:        errorSummary,
				Severity:            classifySeverity(tc),
			}

			result.FailedTests = append(result.FailedTests, info)
		}
	}

	result.ProcessedAt = time.Now()
}

func classifySeverity(tc core.TestCase) core.FailureSeverity {
	if tc.Status == core.TestStatusErrored {
		return core.FailureSeverityCritical
	}

	if tc.Failure == nil {
		return core.FailureSeverityMedium
	}

	return core.FailureSeverityHigh
}

func extractErrorType(failure *core.TestFailure) string {
	if failure.Type != "" {
		return failure.Type
	}

	msg := failure.Message
	if idx := strings.Index(msg, ":"); idx > 0 && idx < 80 {
		candidate := strings.TrimSpace(msg[:idx])
		if !strings.Contains(candidate, " ") || strings.HasSuffix(candidate, "Error") || strings.HasSuffix(candidate, "Exception") {
			return candidate
		}
	}

	return "UnknownError"
}

