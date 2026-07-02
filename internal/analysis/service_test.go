package analysis

import (
	"context"
	"testing"
	"time"

	"sift/internal/core"
)

type callRecordingRepo struct {
	calls        []string
	upsertCount  int
	storedReport *core.TestReport
}

func (r *callRecordingRepo) Store(ctx context.Context, report *core.TestReport) error {
	r.calls = append(r.calls, "store")
	r.storedReport = report
	return nil
}

func (r *callRecordingRepo) GetByID(ctx context.Context, id string) (*core.TestReport, error) {
	return nil, nil
}

func (r *callRecordingRepo) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*core.TestReport, error) {
	return nil, nil
}

func (r *callRecordingRepo) GetFailureHistory(ctx context.Context, testName string, limit int) ([]*core.TestReport, error) {
	return nil, nil
}

func (r *callRecordingRepo) GetTestFailureCounts(ctx context.Context, testName string, since time.Time, excludeReportID string) (int, error) {
	return 0, nil
}

func (r *callRecordingRepo) GetTestLastSuccess(ctx context.Context, testName string) (time.Time, error) {
	return time.Time{}, nil
}

func (r *callRecordingRepo) UpsertErrorFingerprint(ctx context.Context, fingerprint string, normalizedTrace string, timestamp time.Time) error {
	r.calls = append(r.calls, "upsert_fingerprint")
	r.upsertCount++
	return nil
}

func (r *callRecordingRepo) GetErrorFingerprint(ctx context.Context, fingerprint string) (int, time.Time, time.Time, error) {
	return 0, time.Time{}, time.Time{}, nil
}

func (r *callRecordingRepo) Count(ctx context.Context) (int64, error) {
	return 0, nil
}

func (r *callRecordingRepo) Delete(ctx context.Context, id string) error {
	return nil
}

type markerStage struct {
	repo *callRecordingRepo
}

func (s *markerStage) Name() string {
	return "marker"
}

func (s *markerStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	s.repo.calls = append(s.repo.calls, "pipeline")
	result.FailedTests = append(result.FailedTests, core.FailedTestInfo{
		Name:            "testFail",
		Classname:       "Sample",
		NormalizedTrace: "trace",
		HistoricalContext: &core.HistoricalContext{
			ErrorFingerprint: "abc123",
		},
	})
}

func newTestService(repo *callRecordingRepo) *Service {
	pipeline := NewPipeline(&markerStage{repo: repo})
	return NewService(pipeline, repo)
}

func TestAnalyzeRunsPipelineBeforeStore(t *testing.T) {
	repo := &callRecordingRepo{}
	service := newTestService(repo)

	report := &core.TestReport{ID: "r1", Timestamp: time.Now()}
	if _, err := service.Analyze(context.Background(), report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"pipeline", "store", "upsert_fingerprint"}
	if len(repo.calls) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, repo.calls)
	}
	for i, call := range want {
		if repo.calls[i] != call {
			t.Fatalf("expected calls %v, got %v", want, repo.calls)
		}
	}
}

func TestAnalyzePersistsFingerprintsOnce(t *testing.T) {
	repo := &callRecordingRepo{}
	service := newTestService(repo)

	report := &core.TestReport{ID: "r1", Timestamp: time.Now()}
	if _, err := service.Analyze(context.Background(), report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.upsertCount != 1 {
		t.Errorf("expected 1 fingerprint upsert, got %d", repo.upsertCount)
	}
}

func TestReAnalyzeDoesNotWrite(t *testing.T) {
	repo := &callRecordingRepo{}
	service := newTestService(repo)

	report := &core.TestReport{ID: "r1", Timestamp: time.Now()}
	if _, err := service.ReAnalyze(context.Background(), report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, call := range repo.calls {
		if call == "store" || call == "upsert_fingerprint" {
			t.Errorf("expected no writes during re-analysis, got call %q", call)
		}
	}
}
