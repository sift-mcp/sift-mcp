package stages

import (
	"context"
	"testing"

	"sift/internal/core"
)

func TestRootCauseExtractStage_DockerTestcontainers(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testWithDocker",
						Classname: "com.example.DockerTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "Failed to load ApplicationContext",
							StackTrace: `org.springframework.beans.factory.BeanCreationException: Error creating bean
Caused by: org.testcontainers.containers.ContainerLaunchException: Container startup failed
Caused by: org.testcontainers.DockerClientProviderStrategy: Could not find a valid Docker environment`,
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testWithDocker", Classname: "com.example.DockerTest"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	if result.FailedTests[0].CleanedRootCause != "Docker daemon not available (required by Testcontainers)" {
		t.Fatalf("expected Docker root cause, got %q", result.FailedTests[0].CleanedRootCause)
	}
}

func TestRootCauseExtractStage_DataSource(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testDB",
						Classname: "com.example.DBTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "Failed to configure a DataSource: 'url' attribute is not specified",
							StackTrace: `Caused by: org.springframework.boot.autoconfigure.jdbc.DataSourceProperties$DataSourceBeanCreationException: Failed to configure a DataSource
Caused by: java.lang.IllegalStateException: JDBC URL attribute not specified`,
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testDB", Classname: "com.example.DBTest"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	rootCause := result.FailedTests[0].CleanedRootCause
	if rootCause == "" {
		t.Fatal("expected non-empty root cause for DataSource failure")
	}
	if !contains(rootCause, "DataSource not configured") {
		t.Fatalf("expected DataSource root cause, got %q", rootCause)
	}
}

func TestRootCauseExtractStage_BeanCreation(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testBean",
						Classname: "com.example.BeanTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "Failed to load ApplicationContext",
							StackTrace: `Caused by: org.springframework.beans.factory.BeanCreationException: Error creating bean 'myService'
Caused by: java.lang.ClassNotFoundException: com.example.MissingDependency`,
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testBean", Classname: "com.example.BeanTest"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	rootCause := result.FailedTests[0].CleanedRootCause
	if !contains(rootCause, "Spring bean creation failed") {
		t.Fatalf("expected Spring bean creation root cause, got %q", rootCause)
	}
	if !contains(rootCause, "ClassNotFoundException") {
		t.Fatalf("expected innermost cause in root cause, got %q", rootCause)
	}
}

func TestRootCauseExtractStage_InnermostCausedBy(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testChain",
						Classname: "com.example.ChainTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message: "Outer exception",
							StackTrace: `Caused by: java.io.IOException: Something went wrong
Caused by: java.net.SocketException: Connection reset`,
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testChain", Classname: "com.example.ChainTest"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	rootCause := result.FailedTests[0].CleanedRootCause
	if !contains(rootCause, "SocketException") || !contains(rootCause, "Connection reset") {
		t.Fatalf("expected innermost caused-by, got %q", rootCause)
	}
}

func TestRootCauseExtractStage_AssertionFailure(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testAssert",
						Classname: "com.example.AssertTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message:    "expected <42> but was <0>",
							StackTrace: "",
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testAssert", Classname: "com.example.AssertTest"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	rootCause := result.FailedTests[0].CleanedRootCause
	if rootCause == "" {
		t.Fatal("expected non-empty root cause for assertion failure")
	}
	if !contains(rootCause, "expected") {
		t.Fatalf("expected assertion keywords in root cause, got %q", rootCause)
	}
}

func TestRootCauseExtractStage_FallbackToCleanedMessage(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testFallback",
						Classname: "com.example.FallbackTest",
						Status:    core.TestStatusFailed,
						Failure: &core.TestFailure{
							Message:    "Something went wrong in the test",
							StackTrace: "",
						},
					},
				},
			},
		},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testFallback", Classname: "com.example.FallbackTest"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	rootCause := result.FailedTests[0].CleanedRootCause
	if rootCause != "Something went wrong in the test" {
		t.Fatalf("expected cleaned message as fallback, got %q", rootCause)
	}
}

func TestRootCauseExtractStage_NoFailureSkipsGracefully(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{},
	}

	result := &core.AnalysisResult{
		FailedTests: []core.FailedTestInfo{
			{Name: "testMissing", Classname: "com.example.Missing"},
		},
	}

	stage := NewRootCauseExtractStage()
	stage.Process(context.Background(), report, result)

	if result.FailedTests[0].CleanedRootCause != "" {
		t.Fatalf("expected empty root cause when no failure found, got %q", result.FailedTests[0].CleanedRootCause)
	}
}

func TestExtractCausedByChain_MultipleEntries(t *testing.T) {
	text := `java.lang.RuntimeException: Something
Caused by: java.io.IOException: IO failure
Caused by: java.net.ConnectException: Connection refused`

	chain := extractCausedByChain(text)
	if len(chain) != 2 {
		t.Fatalf("expected 2 caused-by entries, got %d", len(chain))
	}
	if !contains(chain[0], "IOException") {
		t.Fatalf("expected first cause to be IOException, got %q", chain[0])
	}
	if !contains(chain[1], "ConnectException") {
		t.Fatalf("expected second cause to be ConnectException, got %q", chain[1])
	}
}

func TestExtractCausedByChain_NoCausedBy(t *testing.T) {
	text := "just a simple error without caused by chain"
	chain := extractCausedByChain(text)
	if len(chain) != 0 {
		t.Fatalf("expected empty chain, got %d entries", len(chain))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
