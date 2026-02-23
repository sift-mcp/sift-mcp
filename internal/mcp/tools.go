package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"sift/internal/core"
)

type Tools struct {
	parser   core.ReportParser
	service  core.AnalysisService
	repo     core.ReportRepository
	readOnly bool
}

func NewTools(parser core.ReportParser, service core.AnalysisService, repo core.ReportRepository, readOnly bool) *Tools {
	return &Tools{
		parser:   parser,
		service:  service,
		repo:     repo,
		readOnly: readOnly,
	}
}

func (t *Tools) RegisterAll(handler *Handler) {
	handler.RegisterTool(t.ingestReportTool(), t.ingestReport)
	handler.RegisterTool(t.analyzeResultsTool(), t.analyzeResults)
	handler.RegisterTool(t.getFailureHistoryTool(), t.getFailureHistory)
	handler.RegisterTool(t.getFlakyTestsTool(), t.getFlakyTests)
	handler.RegisterTool(t.getReportStatsTool(), t.getReportStats)
	handler.RegisterTool(t.getSeverityTrendTool(), t.getSeverityTrend)
}

func (t *Tools) ingestReportTool() Tool {
	return Tool{
		Name:        "ingest_report",
		Description: "Ingest a base64-encoded JUnit XML report. Parses the XML, runs the analysis pipeline, stores in database, and returns the analysis summary. Never returns raw XML.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"report_xml_base64": {
					Type:        "string",
					Description: "Base64-encoded JUnit XML report content",
				},
				"source": {
					Type:        "string",
					Description: "Label for the report source (e.g., 'ci-pipeline', 'local-tests')",
					Default:     "mcp-upload",
				},
			},
			Required: []string{"report_xml_base64"},
		},
	}
}

func (t *Tools) ingestReport(ctx context.Context, args map[string]interface{}) ToolsCallResult {
	b64, ok := args["report_xml_base64"].(string)
	if !ok || b64 == "" {
		return ErrorResult("report_xml_base64 parameter is required")
	}

	xmlData, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid base64 encoding: %v", err))
	}

	report, err := t.parser.Parse(xmlData)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse JUnit XML: %v", err))
	}

	if source, ok := args["source"].(string); ok && source != "" {
		report.Source = source
	}

	result, err := t.service.Analyze(ctx, report)
	if err != nil {
		return ErrorResult(fmt.Sprintf("analysis failed: %v", err))
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolsCallResult{Content: []Content{TextContent(string(data))}}
}

func (t *Tools) analyzeResultsTool() Tool {
	return Tool{
		Name:        "analyze_results",
		Description: "Re-analyze a previously stored test report by its ID. Runs the analysis pipeline again and returns updated results.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"report_id": {
					Type:        "string",
					Description: "UUID of the stored test report to analyze",
				},
			},
			Required: []string{"report_id"},
		},
	}
}

func (t *Tools) analyzeResults(ctx context.Context, args map[string]interface{}) ToolsCallResult {
	reportID, ok := args["report_id"].(string)
	if !ok || reportID == "" {
		return ErrorResult("report_id parameter is required")
	}

	report, err := t.repo.GetByID(ctx, reportID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("report not found: %v", err))
	}

	result, err := t.service.ReAnalyze(ctx, report)
	if err != nil {
		return ErrorResult(fmt.Sprintf("analysis failed: %v", err))
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolsCallResult{Content: []Content{TextContent(string(data))}}
}

func (t *Tools) getFailureHistoryTool() Tool {
	return Tool{
		Name:        "get_failure_history",
		Description: "Get failure history for a specific test name. Shows how many times it failed, when it first and last failed, and related report IDs.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"test_name": {
					Type:        "string",
					Description: "Fully qualified test name to look up",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of historical reports to return",
					Default:     20,
				},
			},
			Required: []string{"test_name"},
		},
	}
}

func (t *Tools) getFailureHistory(ctx context.Context, args map[string]interface{}) ToolsCallResult {
	testName, ok := args["test_name"].(string)
	if !ok || testName == "" {
		return ErrorResult("test_name parameter is required")
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	reports, err := t.repo.GetFailureHistory(ctx, testName, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get failure history: %v", err))
	}

	type failureEntry struct {
		ReportID  string    `json:"report_id"`
		Timestamp time.Time `json:"timestamp"`
		Total     int       `json:"total_tests"`
		Failed    int       `json:"failed"`
		Passed    int       `json:"passed"`
	}

	entries := make([]failureEntry, 0, len(reports))
	for _, r := range reports {
		entries = append(entries, failureEntry{
			ReportID:  r.ID,
			Timestamp: r.Timestamp,
			Total:     r.TotalTests,
			Failed:    r.Failed,
			Passed:    r.Passed,
		})
	}

	result := map[string]interface{}{
		"test_name":     testName,
		"failure_count": len(entries),
		"history":       entries,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolsCallResult{Content: []Content{TextContent(string(data))}}
}

func (t *Tools) getFlakyTestsTool() Tool {
	return Tool{
		Name:        "get_flaky_tests",
		Description: "Identify flaky tests that intermittently pass and fail across stored reports. A test is flaky if it has both passes and failures in recent runs.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"time_range": {
					Type:        "string",
					Description: "Time range to analyze (e.g., 7d, 30d)",
					Default:     "7d",
				},
				"min_runs": {
					Type:        "integer",
					Description: "Minimum number of runs for a test to be considered for flakiness analysis",
					Default:     3,
				},
			},
		},
	}
}

func (t *Tools) getFlakyTests(ctx context.Context, args map[string]interface{}) ToolsCallResult {
	timeRange := "7d"
	if tr, ok := args["time_range"].(string); ok {
		timeRange = tr
	}

	minRuns := 3
	if mr, ok := args["min_runs"].(float64); ok {
		minRuns = int(mr)
	}

	duration, err := parseTimeDuration(timeRange)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid time_range: %v", err))
	}

	end := time.Now()
	start := end.Add(-duration)

	reports, err := t.repo.GetByTimeRange(ctx, start, end)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get reports: %v", err))
	}

	type testKey struct {
		Classname string
		Name      string
	}
	type testStats struct {
		Passes   int
		Failures int
	}
	stats := make(map[testKey]*testStats)

	for _, report := range reports {
		for _, suite := range report.Suites {
			for _, tc := range suite.Cases {
				key := testKey{Classname: tc.Classname, Name: tc.Name}
				if stats[key] == nil {
					stats[key] = &testStats{}
				}
				if tc.Status == core.TestStatusPassed {
					stats[key].Passes++
				} else if tc.Status == core.TestStatusFailed || tc.Status == core.TestStatusErrored {
					stats[key].Failures++
				}
			}
		}
	}

	var flakyTests []core.FlakyTestInfo
	for key, s := range stats {
		totalRuns := s.Passes + s.Failures
		if totalRuns < minRuns || s.Passes == 0 || s.Failures == 0 {
			continue
		}
		flakyTests = append(flakyTests, core.FlakyTestInfo{
			Name:          key.Name,
			Classname:     key.Classname,
			FlakinessRate: float64(s.Failures) / float64(totalRuns) * 100,
			TotalRuns:     totalRuns,
			Failures:      s.Failures,
		})
	}

	sort.Slice(flakyTests, func(i, j int) bool {
		return flakyTests[i].FlakinessRate > flakyTests[j].FlakinessRate
	})

	result := map[string]interface{}{
		"time_range":       timeRange,
		"reports_analyzed": len(reports),
		"flaky_tests":      flakyTests,
		"flaky_count":      len(flakyTests),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolsCallResult{Content: []Content{TextContent(string(data))}}
}

func (t *Tools) getReportStatsTool() Tool {
	return Tool{
		Name:        "get_report_stats",
		Description: "Get overall statistics: total reports stored, aggregate pass/fail rates, and top failing tests.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"time_range": {
					Type:        "string",
					Description: "Time range to analyze (e.g., 7d, 30d, 90d)",
					Default:     "30d",
				},
			},
		},
	}
}

func (t *Tools) getReportStats(ctx context.Context, args map[string]interface{}) ToolsCallResult {
	timeRange := "30d"
	if tr, ok := args["time_range"].(string); ok {
		timeRange = tr
	}

	duration, err := parseTimeDuration(timeRange)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid time_range: %v", err))
	}

	end := time.Now()
	start := end.Add(-duration)

	totalCount, err := t.repo.Count(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to count reports: %v", err))
	}

	reports, err := t.repo.GetByTimeRange(ctx, start, end)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get reports: %v", err))
	}

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	totalErrored := 0
	failureCounts := make(map[string]int)

	for _, r := range reports {
		totalTests += r.TotalTests
		totalPassed += r.Passed
		totalFailed += r.Failed
		totalSkipped += r.Skipped
		totalErrored += r.Errored

		for _, suite := range r.Suites {
			for _, tc := range suite.Cases {
				if tc.Status == core.TestStatusFailed || tc.Status == core.TestStatusErrored {
					key := tc.Classname + "." + tc.Name
					failureCounts[key]++
				}
			}
		}
	}

	type topFailer struct {
		Name  string `json:"name"`
		Count int    `json:"failure_count"`
	}
	var topFailing []topFailer
	for name, count := range failureCounts {
		topFailing = append(topFailing, topFailer{Name: name, Count: count})
	}
	sort.Slice(topFailing, func(i, j int) bool {
		return topFailing[i].Count > topFailing[j].Count
	})
	if len(topFailing) > 10 {
		topFailing = topFailing[:10]
	}

	passRate := float64(0)
	if totalTests > 0 {
		passRate = float64(totalPassed) / float64(totalTests) * 100
	}

	result := map[string]interface{}{
		"total_reports_all_time": totalCount,
		"reports_in_range":      len(reports),
		"time_range":            timeRange,
		"total_tests":           totalTests,
		"total_passed":          totalPassed,
		"total_failed":          totalFailed,
		"total_errored":         totalErrored,
		"total_skipped":         totalSkipped,
		"pass_rate_pct":         fmt.Sprintf("%.2f", passRate),
		"top_failing_tests":     topFailing,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolsCallResult{Content: []Content{TextContent(string(data))}}
}

func (t *Tools) getSeverityTrendTool() Tool {
	return Tool{
		Name:        "get_severity_trend",
		Description: "Get failure severity distribution over time. Shows how critical, high, medium, and low severity failures trend across reports.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"time_range": {
					Type:        "string",
					Description: "Time range to analyze (e.g., 7d, 30d)",
					Default:     "7d",
				},
				"bucket": {
					Type:        "string",
					Description: "Time bucket size (hour, day, week)",
					Default:     "day",
					Enum:        []string{"hour", "day", "week"},
				},
			},
		},
	}
}

func (t *Tools) getSeverityTrend(ctx context.Context, args map[string]interface{}) ToolsCallResult {
	timeRange := "7d"
	if tr, ok := args["time_range"].(string); ok {
		timeRange = tr
	}

	bucket := "day"
	if b, ok := args["bucket"].(string); ok {
		bucket = b
	}

	duration, err := parseTimeDuration(timeRange)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid time_range: %v", err))
	}

	end := time.Now()
	start := end.Add(-duration)

	reports, err := t.repo.GetByTimeRange(ctx, start, end)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get reports: %v", err))
	}

	buckets := make(map[string]map[string]int)

	for _, report := range reports {
		bucketKey := formatBucketKey(report.Timestamp, bucket)

		for _, suite := range report.Suites {
			for _, tc := range suite.Cases {
				if tc.Status != core.TestStatusFailed && tc.Status != core.TestStatusErrored {
					continue
				}
				severity := string(classifySeverityForTrend(tc))
				if buckets[bucketKey] == nil {
					buckets[bucketKey] = make(map[string]int)
				}
				buckets[bucketKey][severity]++
			}
		}
	}

	result := map[string]interface{}{
		"time_range": timeRange,
		"bucket":     bucket,
		"trend":      buckets,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolsCallResult{Content: []Content{TextContent(string(data))}}
}

func parseTimeDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 24 * time.Hour, nil
	}

	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, err
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}

	if strings.HasSuffix(s, "w") {
		weeks := strings.TrimSuffix(s, "w")
		var w int
		if _, err := fmt.Sscanf(weeks, "%d", &w); err != nil {
			return 0, err
		}
		return time.Duration(w) * 7 * 24 * time.Hour, nil
	}

	return time.ParseDuration(s)
}

func formatBucketKey(t time.Time, bucket string) string {
	switch bucket {
	case "hour":
		return t.Format("2006-01-02 15:00")
	case "week":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	default:
		return t.Format("2006-01-02")
	}
}

func classifySeverityForTrend(tc core.TestCase) core.FailureSeverity {
	if tc.Status == core.TestStatusErrored {
		return core.FailureSeverityCritical
	}
	if tc.Failure == nil {
		return core.FailureSeverityMedium
	}
	return core.FailureSeverityHigh
}
