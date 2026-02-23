package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"sift/internal/core"
)

type mockParser struct{}

func (m *mockParser) Parse(data []byte) (*core.TestReport, error) {
	return &core.TestReport{
		ID:         "test-report-001",
		Timestamp:  time.Now(),
		Source:     "test",
		Framework:  "junit",
		TotalTests: 3,
		Passed:     1,
		Failed:     1,
		Errored:    1,
		Skipped:    0,
		Suites: []core.TestSuite{
			{
				Name:     "SampleSuite",
				Tests:    3,
				Failures: 1,
				Errors:   1,
				Cases: []core.TestCase{
					{Name: "testPass", Classname: "Sample", Status: core.TestStatusPassed},
					{
						Name: "testFail", Classname: "Sample", Status: core.TestStatusFailed,
						Failure: &core.TestFailure{Message: "assertion failed", StackTrace: "at line 10"},
					},
					{
						Name: "testError", Classname: "Sample", Status: core.TestStatusErrored,
						Failure: &core.TestFailure{Message: "null pointer", StackTrace: "at line 20"},
					},
				},
			},
		},
	}, nil
}

func (m *mockParser) ParseStream(reader io.Reader) (*core.TestReport, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return m.Parse(data)
}

func (m *mockParser) Format() string { return "mock" }

type mockService struct {
	lastReport *core.TestReport
}

func (m *mockService) Analyze(ctx context.Context, report *core.TestReport) (*core.AnalysisResult, error) {
	m.lastReport = report
	return &core.AnalysisResult{
		ReportID: report.ID,
		FailedTests: []core.FailedTestInfo{
			{Name: "testFail", Classname: "Sample", Severity: core.FailureSeverityHigh, ErrorClassification: "AssertionError", ErrorSummary: "assertion failed"},
			{Name: "testError", Classname: "Sample", Severity: core.FailureSeverityCritical, ErrorClassification: "NullPointerException", ErrorSummary: "null pointer"},
		},
		Summary:     "2/3 tests failed",
		ProcessedAt: time.Now(),
	}, nil
}

func (m *mockService) ReAnalyze(ctx context.Context, report *core.TestReport) (*core.AnalysisResult, error) {
	return m.Analyze(ctx, report)
}

type mockRepo struct {
	storedReports map[string]*core.TestReport
}

func newMockRepo() *mockRepo {
	return &mockRepo{storedReports: make(map[string]*core.TestReport)}
}

func (m *mockRepo) Store(ctx context.Context, report *core.TestReport) error {
	m.storedReports[report.ID] = report
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*core.TestReport, error) {
	if r, ok := m.storedReports[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockRepo) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*core.TestReport, error) {
	var results []*core.TestReport
	for _, r := range m.storedReports {
		if !r.Timestamp.Before(start) && !r.Timestamp.After(end) {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *mockRepo) GetFailureHistory(ctx context.Context, testName string, limit int) ([]*core.TestReport, error) {
	var results []*core.TestReport
	for _, r := range m.storedReports {
		for _, s := range r.Suites {
			for _, tc := range s.Cases {
				if tc.Name == testName && (tc.Status == core.TestStatusFailed || tc.Status == core.TestStatusErrored) {
					results = append(results, r)
				}
			}
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *mockRepo) GetTestFailureCounts(ctx context.Context, testName string, since time.Time) (int, error) {
	count := 0
	for _, r := range m.storedReports {
		if r.Timestamp.Before(since) {
			continue
		}
		for _, s := range r.Suites {
			for _, tc := range s.Cases {
				if tc.Name == testName && (tc.Status == core.TestStatusFailed || tc.Status == core.TestStatusErrored) {
					count++
				}
			}
		}
	}
	return count, nil
}

func (m *mockRepo) GetTestLastSuccess(ctx context.Context, testName string) (time.Time, error) {
	return time.Time{}, nil
}

func (m *mockRepo) UpsertErrorFingerprint(ctx context.Context, fingerprint string, normalizedTrace string, timestamp time.Time) error {
	return nil
}

func (m *mockRepo) GetErrorFingerprint(ctx context.Context, fingerprint string) (int, time.Time, time.Time, error) {
	return 0, time.Time{}, time.Time{}, nil
}

func (m *mockRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.storedReports)), nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	delete(m.storedReports, id)
	return nil
}

func setupTools() (*Tools, *mockRepo) {
	repo := newMockRepo()
	parser := &mockParser{}
	service := &mockService{}
	tools := NewTools(parser, service, repo, false)
	return tools, repo
}

func TestIngestReport(t *testing.T) {
	tools, _ := setupTools()
	ctx := context.Background()

	sampleXML := `<testsuite name="test" tests="1"><testcase name="t1" classname="C"/></testsuite>`
	b64 := base64.StdEncoding.EncodeToString([]byte(sampleXML))

	result := tools.ingestReport(ctx, map[string]interface{}{
		"report_xml_base64": b64,
		"source":            "ci-test",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var analysisResult map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &analysisResult); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if analysisResult["report_id"] == nil {
		t.Error("expected report_id in response")
	}
	if analysisResult["summary"] == nil {
		t.Error("expected summary in response")
	}
}

func TestIngestReportMissingBase64(t *testing.T) {
	tools, _ := setupTools()
	ctx := context.Background()

	result := tools.ingestReport(ctx, map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error for missing base64 parameter")
	}
}

func TestIngestReportInvalidBase64(t *testing.T) {
	tools, _ := setupTools()
	ctx := context.Background()

	result := tools.ingestReport(ctx, map[string]interface{}{
		"report_xml_base64": "not-valid-base64!!!",
	})
	if !result.IsError {
		t.Error("expected error for invalid base64")
	}
}

func TestGetReportStats(t *testing.T) {
	tools, repo := setupTools()
	ctx := context.Background()

	repo.storedReports["r1"] = &core.TestReport{
		ID: "r1", Timestamp: time.Now(), TotalTests: 10, Passed: 8, Failed: 2,
		Suites: []core.TestSuite{
			{Cases: []core.TestCase{
				{Name: "a", Classname: "X", Status: core.TestStatusFailed},
				{Name: "b", Classname: "X", Status: core.TestStatusFailed},
			}},
		},
	}

	result := tools.getReportStats(ctx, map[string]interface{}{"time_range": "30d"})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var stats map[string]interface{}
	json.Unmarshal([]byte(result.Content[0].Text), &stats)

	if stats["total_reports_all_time"].(float64) != 1 {
		t.Errorf("expected 1 total report, got %v", stats["total_reports_all_time"])
	}
}

func TestGetFailureHistory(t *testing.T) {
	tools, repo := setupTools()
	ctx := context.Background()

	repo.storedReports["r1"] = &core.TestReport{
		ID: "r1", Timestamp: time.Now(), TotalTests: 2, Passed: 1, Failed: 1,
		Suites: []core.TestSuite{
			{Cases: []core.TestCase{
				{Name: "testFail", Classname: "X", Status: core.TestStatusFailed},
			}},
		},
	}

	result := tools.getFailureHistory(ctx, map[string]interface{}{
		"test_name": "testFail",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var history map[string]interface{}
	json.Unmarshal([]byte(result.Content[0].Text), &history)

	if history["failure_count"].(float64) != 1 {
		t.Errorf("expected 1 failure, got %v", history["failure_count"])
	}
}

func TestGetFailureHistoryMissingName(t *testing.T) {
	tools, _ := setupTools()
	ctx := context.Background()

	result := tools.getFailureHistory(ctx, map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error for missing test_name")
	}
}

func TestParseTimeDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1h", time.Hour},
		{"24h", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"2w", 14 * 24 * time.Hour},
		{"30m", 30 * time.Minute},
	}

	for _, tc := range tests {
		d, err := parseTimeDuration(tc.input)
		if err != nil {
			t.Errorf("parseTimeDuration(%q) error: %v", tc.input, err)
			continue
		}
		if d != tc.expected {
			t.Errorf("parseTimeDuration(%q) = %v, want %v", tc.input, d, tc.expected)
		}
	}
}

func TestFormatBucketKey(t *testing.T) {
	ts := time.Date(2026, 2, 12, 15, 30, 0, 0, time.UTC)

	if got := formatBucketKey(ts, "hour"); got != "2026-02-12 15:00" {
		t.Errorf("hour bucket = %q, want '2026-02-12 15:00'", got)
	}
	if got := formatBucketKey(ts, "day"); got != "2026-02-12" {
		t.Errorf("day bucket = %q, want '2026-02-12'", got)
	}
	if got := formatBucketKey(ts, "week"); got != "2026-W07" {
		t.Errorf("week bucket = %q, want '2026-W07'", got)
	}
}
