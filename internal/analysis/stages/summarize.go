package stages

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"sift/internal/core"
)

type SummarizeStage struct {
	repo core.ReportRepository
}

func NewSummarizeStage(repo core.ReportRepository) *SummarizeStage {
	return &SummarizeStage{repo: repo}
}

func (s *SummarizeStage) Name() string {
	return "summarize"
}

func (s *SummarizeStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	result.FailureGroups = s.groupByRootCause(result.FailedTests)
	result.CascadeSummary = s.computeCascadeSummary(result.FailedTests)
	result.Delta = s.computeDelta(ctx, report, result.FailedTests)
	result.Summary = s.buildSummary(report, result)
}

func (s *SummarizeStage) groupingKey(f core.FailedTestInfo) string {
	if f.CleanedRootCause != "" {
		return f.CleanedRootCause + "|" + f.ErrorClassification
	}
	return f.ErrorSummary + "|" + f.ErrorClassification
}

func (s *SummarizeStage) groupByRootCause(failures []core.FailedTestInfo) []core.FailureGroup {
	if len(failures) == 0 {
		return []core.FailureGroup{}
	}

	type groupAccumulator struct {
		fingerprint         string
		rootCause           string
		errorClassification string
		errorSummary        string
		category            core.FailureCategory
		testNames           []string
		suiteCountMap       map[string]int
		firstSeen           time.Time
		lastSeen            time.Time
		isFlaky             bool
		originalCount       int
		cascadeCount        int
	}

	grouped := make(map[string]*groupAccumulator)
	insertOrder := make([]string, 0)

	for _, f := range failures {
		key := s.groupingKey(f)

		acc, exists := grouped[key]
		if !exists {
			rootCause := f.CleanedRootCause
			if rootCause == "" {
				rootCause = f.ErrorSummary
			}

			fingerprint := "unknown"
			if f.HistoricalContext != nil && f.HistoricalContext.ErrorFingerprint != "" {
				fingerprint = f.HistoricalContext.ErrorFingerprint
			}

			acc = &groupAccumulator{
				fingerprint:         fingerprint,
				rootCause:           rootCause,
				errorClassification: f.ErrorClassification,
				errorSummary:        f.ErrorSummary,
				category:            classifyCategory(f.ErrorClassification, rootCause),
				suiteCountMap:       make(map[string]int),
			}
			grouped[key] = acc
			insertOrder = append(insertOrder, key)
		}

		acc.testNames = append(acc.testNames, f.Classname+"."+f.Name)
		acc.suiteCountMap[f.Classname]++

		if f.IsCascade {
			acc.cascadeCount++
		} else {
			acc.originalCount++
		}

		if f.HistoricalContext != nil {
			if !f.HistoricalContext.FirstSeen.IsZero() && (acc.firstSeen.IsZero() || f.HistoricalContext.FirstSeen.Before(acc.firstSeen)) {
				acc.firstSeen = f.HistoricalContext.FirstSeen
			}
			if f.HistoricalContext.LastSeen.After(acc.lastSeen) {
				acc.lastSeen = f.HistoricalContext.LastSeen
			}
			if f.HistoricalContext.IsFlaky {
				acc.isFlaky = true
			}
		}
	}

	groups := make([]core.FailureGroup, 0, len(grouped))
	for _, key := range insertOrder {
		acc := grouped[key]

		suitesList := make([]string, 0, len(acc.suiteCountMap))
		for suite, count := range acc.suiteCountMap {
			shortName := suite
			if idx := strings.LastIndex(suite, "."); idx >= 0 {
				shortName = suite[idx+1:]
			}
			suitesList = append(suitesList, fmt.Sprintf("%s (%d)", shortName, count))
		}
		sort.Strings(suitesList)

		groups = append(groups, core.FailureGroup{
			RootCause:            acc.rootCause,
			ErrorClassification:  acc.errorClassification,
			Fingerprint:          acc.fingerprint,
			Category:             acc.category,
			AffectedTests:        len(acc.testNames),
			OriginalFailureCount: acc.originalCount,
			CascadeFailureCount:  acc.cascadeCount,
			AffectedSuites:       suitesList,
			SampleMessage:        acc.errorSummary,
			FirstSeen:            acc.firstSeen,
			LastSeen:             acc.lastSeen,
			IsFlaky:              acc.isFlaky,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].AffectedTests > groups[j].AffectedTests
	})

	return groups
}

func (s *SummarizeStage) computeCascadeSummary(failures []core.FailedTestInfo) *core.CascadeSummary {
	totalOriginal := 0
	totalCascade := 0

	for _, f := range failures {
		if f.IsCascade {
			totalCascade++
		} else {
			totalOriginal++
		}
	}

	if totalCascade == 0 {
		return nil
	}

	total := totalOriginal + totalCascade
	cascadePercentage := 0.0
	if total > 0 {
		cascadePercentage = float64(totalCascade) / float64(total) * 100.0
	}

	return &core.CascadeSummary{
		TotalOriginalFailures: totalOriginal,
		TotalCascadeFailures:  totalCascade,
		CascadePercentage:     cascadePercentage,
	}
}

func classifyCategory(errorClassification string, errorSummary string) core.FailureCategory {
	lowerClass := strings.ToLower(errorClassification)
	lowerSummary := strings.ToLower(errorSummary)

	infrastructurePatterns := []string{
		"docker", "container", "testcontainers",
		"connection refused", "connection timed out", "connect to",
		"datasource", "failed to load applicationcontext",
		"applicationcontext failure threshold",
		"beancreationexception",
		"no such file or directory",
		"permission denied",
		"out of memory", "oom",
		"socket", "bind address",
		"spring bean creation failed",
	}

	for _, pattern := range infrastructurePatterns {
		if strings.Contains(lowerClass, pattern) || strings.Contains(lowerSummary, pattern) {
			return core.FailureCategoryInfrastructure
		}
	}

	assertionPatterns := []string{
		"assert", "expected", "junit",
		"comparisonf", "equalsf",
	}

	for _, pattern := range assertionPatterns {
		if strings.Contains(lowerClass, pattern) || strings.Contains(lowerSummary, pattern) {
			return core.FailureCategoryAssertion
		}
	}

	return core.FailureCategoryCode
}

func (s *SummarizeStage) computeDelta(ctx context.Context, report *core.TestReport, currentFailures []core.FailedTestInfo) *core.RunDelta {
	if s.repo == nil {
		return nil
	}

	end := report.Timestamp.Add(-1 * time.Second)
	start := end.Add(-30 * 24 * time.Hour)

	previousReports, err := s.repo.GetByTimeRange(ctx, start, end)
	if err != nil || len(previousReports) == 0 {
		return &core.RunDelta{
			NewFailures:    len(currentFailures),
			FixedSinceLast: 0,
			Recurring:      0,
		}
	}

	previousReport := previousReports[0]

	previousFailureSet := make(map[string]bool)
	for _, suite := range previousReport.Suites {
		for _, tc := range suite.Cases {
			if tc.Status == core.TestStatusFailed || tc.Status == core.TestStatusErrored {
				previousFailureSet[tc.Classname+"."+tc.Name] = true
			}
		}
	}

	currentFailureSet := make(map[string]bool)
	for _, f := range currentFailures {
		currentFailureSet[f.Classname+"."+f.Name] = true
	}

	newFailures := 0
	recurring := 0
	for key := range currentFailureSet {
		if previousFailureSet[key] {
			recurring++
		} else {
			newFailures++
		}
	}

	fixedSinceLast := 0
	for key := range previousFailureSet {
		if !currentFailureSet[key] {
			fixedSinceLast++
		}
	}

	return &core.RunDelta{
		NewFailures:    newFailures,
		FixedSinceLast: fixedSinceLast,
		Recurring:      recurring,
	}
}

func (s *SummarizeStage) buildSummary(report *core.TestReport, result *core.AnalysisResult) string {
	totalFailed := report.Failed + report.Errored

	if totalFailed == 0 {
		return fmt.Sprintf("All %d tests passed. No failures detected.", report.TotalTests)
	}

	groupCount := len(result.FailureGroups)

	infraCount := 0
	assertionCount := 0
	codeCount := 0
	for _, g := range result.FailureGroups {
		switch g.Category {
		case core.FailureCategoryInfrastructure:
			infraCount += g.AffectedTests
		case core.FailureCategoryAssertion:
			assertionCount += g.AffectedTests
		case core.FailureCategoryCode:
			codeCount += g.AffectedTests
		}
	}

	cascadePart := ""
	if result.CascadeSummary != nil {
		cascadePart = fmt.Sprintf(" (%d original, %d cascading)",
			result.CascadeSummary.TotalOriginalFailures,
			result.CascadeSummary.TotalCascadeFailures)
	}

	parts := []string{
		fmt.Sprintf("%d/%d tests failed from %d root causes%s.", totalFailed, report.TotalTests, groupCount, cascadePart),
	}

	categoryParts := make([]string, 0)
	if infraCount > 0 {
		categoryParts = append(categoryParts, fmt.Sprintf("%d infrastructure", infraCount))
	}
	if assertionCount > 0 {
		categoryParts = append(categoryParts, fmt.Sprintf("%d assertion", assertionCount))
	}
	if codeCount > 0 {
		categoryParts = append(categoryParts, fmt.Sprintf("%d code", codeCount))
	}

	if len(categoryParts) > 0 {
		parts = append(parts, fmt.Sprintf("Categories: %s.", strings.Join(categoryParts, ", ")))
	}

	if result.Delta != nil && result.Delta.NewFailures > 0 {
		parts = append(parts, fmt.Sprintf("%d new since last run.", result.Delta.NewFailures))
	}
	if result.Delta != nil && result.Delta.FixedSinceLast > 0 {
		parts = append(parts, fmt.Sprintf("%d fixed since last run.", result.Delta.FixedSinceLast))
	}

	return strings.Join(parts, " ")
}
