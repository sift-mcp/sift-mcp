package stages

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"sift/internal/core"
)

var normalizationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`0x[0-9a-fA-F]+`),
	regexp.MustCompile(`@[0-9a-fA-F]{4,16}\b`),
	regexp.MustCompile(`(?i)for \[MergedContextConfiguration[^\n]*`),
	regexp.MustCompile(`(?i)\btestClass\s*=\s*[^\s,\]]+`),
	regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[^\s]*`),
	regexp.MustCompile(`\b(Thread|thread|tid)[- ]?\d+\b`),
	regexp.MustCompile(`\b\d{10,13}\b`),
	regexp.MustCompile(`:[0-9]+\)`),
	regexp.MustCompile(`\s+`),
}

type FingerprintStage struct{}

func NewFingerprintStage() *FingerprintStage {
	return &FingerprintStage{}
}

func (s *FingerprintStage) Name() string {
	return "fingerprint"
}

func (s *FingerprintStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	for i := range result.FailedTests {
		failedTest := &result.FailedTests[i]

		stackTrace := ""
		failure := lookupTestFailure(report, failedTest.Name, failedTest.Classname)
		if failure != nil {
			stackTrace = failure.StackTrace
		}

		normalized := NormalizeStackTrace(stackTrace)

		if failedTest.HistoricalContext == nil {
			failedTest.HistoricalContext = &core.HistoricalContext{}
		}
		failedTest.HistoricalContext.ErrorFingerprint = HashFingerprint(normalized)
		failedTest.NormalizedTrace = normalized
	}
}

func NormalizeStackTrace(trace string) string {
	if trace == "" {
		return ""
	}

	normalized := trace
	for _, pattern := range normalizationPatterns {
		normalized = pattern.ReplaceAllString(normalized, "")
	}

	normalized = strings.TrimSpace(normalized)
	normalized = strings.ToLower(normalized)
	return normalized
}

func HashFingerprint(normalizedTrace string) string {
	hash := sha256.Sum256([]byte(normalizedTrace))
	return fmt.Sprintf("%x", hash[:8])
}
