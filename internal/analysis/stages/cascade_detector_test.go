package stages

import (
	"context"
	"testing"

	"sift/internal/core"
)

func TestCascadeDetectStage_IdentifiesCascadeByThresholdMessage(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testOriginal",
						Classname: "com.example.MyTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "Failed to load ApplicationContext",
						},
					},
					{
						Name:      "testCascade",
						Classname: "com.example.MyTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "ApplicationContext failure threshold (1) exceeded: skipping repeated attempt to load context",
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{
				Name:             "testOriginal",
				Classname:        "com.example.MyTest",
				CleanedRootCause: "Docker daemon not available (required by Testcontainers)",
			},
			{
				Name:      "testCascade",
				Classname: "com.example.MyTest",
			},
		},
	}

	stage := NewCascadeDetectStage()
	stage.Process(context.Background(), report, result)

	if result.FailedTests[0].IsCascade {
		t.Fatal("original failure should not be marked as cascade")
	}
	if !result.FailedTests[1].IsCascade {
		t.Fatal("cascade failure should be marked as cascade")
	}
	if result.FailedTests[1].CascadeSourceTest != "com.example.MyTest" {
		t.Fatalf("expected cascade source to be com.example.MyTest, got %q", result.FailedTests[1].CascadeSourceTest)
	}
}

func TestCascadeDetectStage_PropagatesCleanedRootCause(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testOriginal",
						Classname: "com.example.MyTest",
						Status:    core.TestStatusFailed,
						Failure:   &core.TestFailure{Message: "Docker not available"},
					},
					{
						Name:      "testCascade",
						Classname: "com.example.MyTest",
						Status:    core.TestStatusFailed,
						Failure:   &core.TestFailure{Message: "ApplicationContext failure threshold exceeded: skipping repeated attempt"},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{
				Name:             "testOriginal",
				Classname:        "com.example.MyTest",
				CleanedRootCause: "Docker daemon not available (required by Testcontainers)",
			},
			{
				Name:      "testCascade",
				Classname: "com.example.MyTest",
			},
		},
	}

	stage := NewCascadeDetectStage()
	stage.Process(context.Background(), report, result)

	if result.FailedTests[1].CleanedRootCause != "Docker daemon not available (required by Testcontainers)" {
		t.Fatalf("expected cascade to inherit root cause, got %q", result.FailedTests[1].CleanedRootCause)
	}
}

func TestCascadeDetectStage_NoCascadesWhenNonePresent(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testA",
						Classname: "com.example.TestA",
						Status:    core.TestStatusFailed,
						Failure:   &core.TestFailure{Message: "assertion failed: expected 1 but got 2"},
					},
					{
						Name:      "testB",
						Classname: "com.example.TestB",
						Status:    core.TestStatusFailed,
						Failure:   &core.TestFailure{Message: "null pointer exception"},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testA", Classname: "com.example.TestA"},
			{Name: "testB", Classname: "com.example.TestB"},
		},
	}

	stage := NewCascadeDetectStage()
	stage.Process(context.Background(), report, result)

	for _, f := range result.FailedTests {
		if f.IsCascade {
			t.Fatalf("no failures should be marked as cascade, but %s was", f.Name)
		}
	}
}

func TestCascadeDetectStage_CrossClassPackageMatch(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testOriginal",
						Classname: "com.example.pkg.OriginalTest",
						Status:    core.TestStatusFailed,
						Failure:   &core.TestFailure{Message: "Docker not available"},
					},
					{
						Name:      "testCascade",
						Classname: "com.example.pkg.OtherTest",
						Status:    core.TestStatusFailed,
						Failure:   &core.TestFailure{Message: "ApplicationContext failure threshold exceeded: skipping repeated attempt"},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{
				Name:             "testOriginal",
				Classname:        "com.example.pkg.OriginalTest",
				CleanedRootCause: "Docker daemon not available (required by Testcontainers)",
			},
			{
				Name:      "testCascade",
				Classname: "com.example.pkg.OtherTest",
			},
		},
	}

	stage := NewCascadeDetectStage()
	stage.Process(context.Background(), report, result)

	if !result.FailedTests[1].IsCascade {
		t.Fatal("expected cross-class cascade to be detected via package prefix match")
	}
	if result.FailedTests[1].CascadeSourceTest != "com.example.pkg.OriginalTest" {
		t.Fatalf("expected cascade source from package match, got %q", result.FailedTests[1].CascadeSourceTest)
	}
}

func TestExtractPackagePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"com.example.pkg.MyTest", "com.example.pkg"},
		{"MyTest", "MyTest"},
		{"com.MyTest", "com"},
	}

	for _, tc := range tests {
		result := extractPackagePrefix(tc.input)
		if result != tc.expected {
			t.Fatalf("extractPackagePrefix(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestIsCascadeFailure_DetectsThresholdPattern(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testA",
						Classname: "com.example.Test",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "ApplicationContext failure threshold (1) exceeded: skipping repeated attempt to load context for [MergedContextConfiguration...]",
						},
					},
				},
			},
		},
	}

	failedTest := &core.FailedTestInfo{Name: "testA", Classname: "com.example.Test"}
	if !isCascadeFailure(report, failedTest) {
		t.Fatal("expected cascade pattern to be detected")
	}
}

func TestIsCascadeFailure_NonCascadeMessage(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testA",
						Classname: "com.example.Test",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "expected true but got false",
						},
					},
				},
			},
		},
	}

	failedTest := &core.FailedTestInfo{Name: "testA", Classname: "com.example.Test"}
	if isCascadeFailure(report, failedTest) {
		t.Fatal("regular assertion failure should not be detected as cascade")
	}
}
