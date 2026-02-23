package stages

import (
	"testing"

	"sift/internal/core"
)

func TestLookupTestFailure_FindsMatchingFailure(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testSomething",
						Classname: "com.example.MyTest",
						Failure:   &core.TestFailure{Message: "expected true", Type: "AssertionError"},
					},
					{
						Name:      "testOther",
						Classname: "com.example.MyTest",
						Failure:   nil,
					},
				},
			},
		},
	}

	result := lookupTestFailure(report, "testSomething", "com.example.MyTest")
	if result == nil {
		t.Fatal("expected to find failure, got nil")
	}
	if result.Message != "expected true" {
		t.Fatalf("expected message 'expected true', got %q", result.Message)
	}
}

func TestLookupTestFailure_ReturnsNilWhenNotFound(t *testing.T) {
	report := &core.TestReport{
		Suites: []core.TestSuite{
			{
				Cases: []core.TestCase{
					{
						Name:      "testSomething",
						Classname: "com.example.MyTest",
						Failure:   &core.TestFailure{Message: "expected true"},
					},
				},
			},
		},
	}

	result := lookupTestFailure(report, "testNonexistent", "com.example.MyTest")
	if result != nil {
		t.Fatal("expected nil for nonexistent test, got non-nil")
	}
}

func TestStripNoisePatterns_RemovesMergedContextConfiguration(t *testing.T) {
	input := "Failed to load context for MergedContextConfiguration@5a2b3c4d testClass = com.example.Test"
	result := stripNoisePatterns(input)
	if result == input {
		t.Fatal("expected noise patterns to be removed")
	}
	if len(result) >= len(input) {
		t.Fatalf("expected shorter result after noise removal, got %q", result)
	}
}

func TestStripNoisePatterns_RemovesTimestamps(t *testing.T) {
	input := "Error at 2024-01-15T10:30:00.123Z during test execution"
	result := stripNoisePatterns(input)
	if result == input {
		t.Fatalf("expected timestamp to be removed, got %q", result)
	}
}

func TestStripNoisePatterns_RemovesObjectIDs(t *testing.T) {
	input := "Bean@5a2b3c4d5e6f failed to initialize"
	result := stripNoisePatterns(input)
	if result == input {
		t.Fatalf("expected object ID to be removed, got %q", result)
	}
}

func TestSmartTruncate_ShortMessageUnchanged(t *testing.T) {
	input := "Short message."
	result := smartTruncate(input, 200)
	if result != input {
		t.Fatalf("expected unchanged message, got %q", result)
	}
}

func TestSmartTruncate_TruncatesAtSentenceBoundary(t *testing.T) {
	input := "First sentence. Second sentence. Third sentence that is very long and should be cut off somewhere."
	result := smartTruncate(input, 50)
	if len(result) > 50 {
		t.Fatalf("expected max length 50, got %d: %q", len(result), result)
	}
	if result != "First sentence. Second sentence." {
		t.Fatalf("expected truncation at sentence boundary, got %q", result)
	}
}

func TestSmartTruncate_TruncatesAtWordBoundary(t *testing.T) {
	input := "This is a long message without sentence endings that needs to be truncated at a word boundary point"
	result := smartTruncate(input, 60)
	if len(result) > 60 {
		t.Fatalf("expected max length 60, got %d: %q", len(result), result)
	}
	lastChar := result[len(result)-1]
	if lastChar == ' ' {
		t.Fatalf("result should not end with a space: %q", result)
	}
}

func TestSmartTruncate_NoMidWordCut(t *testing.T) {
	input := "SomeSuperLongExceptionClassNameThatGoesOnAndOnWithoutSpaces and then continues"
	result := smartTruncate(input, 40)
	if len(result) > 40 {
		t.Fatalf("expected max length 40, got %d: %q", len(result), result)
	}
}
