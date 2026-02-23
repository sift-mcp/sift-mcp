package core

import (
	"context"
	"io"
	"time"
)

type TestStatus string

const (
	TestStatusPassed  TestStatus = "passed"
	TestStatusFailed  TestStatus = "failed"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusErrored TestStatus = "errored"
)

type FailureSeverity string

const (
	FailureSeverityCritical FailureSeverity = "critical"
	FailureSeverityHigh     FailureSeverity = "high"
	FailureSeverityMedium   FailureSeverity = "medium"
	FailureSeverityLow      FailureSeverity = "low"
)

type TestReport struct {
	ID         string        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	Source     string        `json:"source"`
	Framework  string        `json:"framework"`
	TotalTests int           `json:"total_tests"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Skipped    int           `json:"skipped"`
	Errored    int           `json:"errored"`
	Duration   time.Duration `json:"duration"`
	Suites     []TestSuite   `json:"suites"`
}

type TestSuite struct {
	Name      string        `json:"name"`
	Package   string        `json:"package"`
	Tests     int           `json:"tests"`
	Failures  int           `json:"failures"`
	Errors    int           `json:"errors"`
	Skipped   int           `json:"skipped"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Cases     []TestCase    `json:"cases"`
}

type TestCase struct {
	Name      string        `json:"name"`
	Classname string        `json:"classname"`
	Status    TestStatus    `json:"status"`
	Duration  time.Duration `json:"duration"`
	Failure   *TestFailure  `json:"failure,omitempty"`
}

type TestFailure struct {
	Message    string `json:"message"`
	Type       string `json:"type"`
	StackTrace string `json:"stack_trace"`
}

type FailureCategory string

const (
	FailureCategoryInfrastructure FailureCategory = "infrastructure"
	FailureCategoryAssertion      FailureCategory = "assertion"
	FailureCategoryCode           FailureCategory = "code"
)

type CascadeSummary struct {
	TotalOriginalFailures int     `json:"total_original_failures"`
	TotalCascadeFailures  int     `json:"total_cascade_failures"`
	CascadePercentage     float64 `json:"cascade_percentage"`
}

type AnalysisResult struct {
	ReportID       string           `json:"report_id"`
	TotalTests     int              `json:"total_tests"`
	Failed         int              `json:"failed"`
	Passed         int              `json:"passed"`
	Skipped        int              `json:"skipped"`
	FailureGroups  []FailureGroup   `json:"failure_groups"`
	Delta          *RunDelta        `json:"delta,omitempty"`
	CascadeSummary *CascadeSummary  `json:"cascade_summary,omitempty"`
	Summary        string           `json:"summary"`
	ProcessedAt    time.Time        `json:"processed_at"`

	FailedTests []FailedTestInfo `json:"-"`
}

type FailureGroup struct {
	RootCause            string          `json:"root_cause"`
	ErrorClassification  string          `json:"error_classification"`
	Fingerprint          string          `json:"fingerprint"`
	Category             FailureCategory `json:"category"`
	AffectedTests        int             `json:"affected_tests"`
	OriginalFailureCount int             `json:"original_failure_count"`
	CascadeFailureCount  int             `json:"cascade_failure_count"`
	AffectedSuites       []string        `json:"affected_suites"`
	SampleMessage        string          `json:"sample_message,omitempty"`
	FirstSeen            time.Time       `json:"first_seen,omitempty"`
	LastSeen             time.Time       `json:"last_seen,omitempty"`
	IsFlaky              bool            `json:"is_flaky"`
}

type RunDelta struct {
	NewFailures    int `json:"new_failures"`
	FixedSinceLast int `json:"fixed_since_last"`
	Recurring      int `json:"recurring"`
}

type FailedTestInfo struct {
	Name                string            `json:"name"`
	Classname           string            `json:"classname"`
	ErrorClassification string            `json:"error_classification"`
	ErrorSummary        string            `json:"error_summary"`
	Severity            FailureSeverity   `json:"severity"`
	HistoricalContext   *HistoricalContext `json:"historical_context,omitempty"`
	CleanedRootCause    string            `json:"-"`
	IsCascade           bool              `json:"-"`
	CascadeSourceTest   string            `json:"-"`
}

type HistoricalContext struct {
	FailureCount24h  int       `json:"failure_count_24h"`
	FailureCount7d   int       `json:"failure_count_7d"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	IsFlaky          bool      `json:"is_flaky"`
	ErrorFingerprint string    `json:"error_fingerprint"`
}

type FlakyTestInfo struct {
	Name          string  `json:"name"`
	Classname     string  `json:"classname"`
	FlakinessRate float64 `json:"flakiness_rate"`
	TotalRuns     int     `json:"total_runs"`
	Failures      int     `json:"failures"`
}

type ReportParser interface {
	Parse(data []byte) (*TestReport, error)
	ParseStream(reader io.Reader) (*TestReport, error)
	Format() string
}

type ReportRepository interface {
	Store(ctx context.Context, report *TestReport) error
	GetByID(ctx context.Context, id string) (*TestReport, error)
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*TestReport, error)
	GetFailureHistory(ctx context.Context, testName string, limit int) ([]*TestReport, error)
	GetTestFailureCounts(ctx context.Context, testName string, since time.Time) (int, error)
	GetTestLastSuccess(ctx context.Context, testName string) (time.Time, error)
	UpsertErrorFingerprint(ctx context.Context, fingerprint string, normalizedTrace string, timestamp time.Time) error
	GetErrorFingerprint(ctx context.Context, fingerprint string) (int, time.Time, time.Time, error)
	Count(ctx context.Context) (int64, error)
	Delete(ctx context.Context, id string) error
}

type AnalysisService interface {
	Analyze(ctx context.Context, report *TestReport) (*AnalysisResult, error)
	ReAnalyze(ctx context.Context, report *TestReport) (*AnalysisResult, error)
}
