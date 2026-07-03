package parser

import (
	"testing"
	"time"

	"sift/internal/core"
)

var multiSuiteXML = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="5" failures="2" errors="1" skipped="1" time="4.5">
  <testsuite name="AuthTest" package="com.app.auth" tests="3" failures="1" errors="1" skipped="0" time="2.5">
    <testcase name="testLogin" classname="com.app.auth.AuthTest" time="0.5"/>
    <testcase name="testLogout" classname="com.app.auth.AuthTest" time="0.8">
      <failure message="Expected redirect" type="AssertionError">stack trace here</failure>
    </testcase>
    <testcase name="testSession" classname="com.app.auth.AuthTest" time="1.2">
      <error message="NullPointerException" type="java.lang.NullPointerException">null ref at line 42</error>
    </testcase>
  </testsuite>
  <testsuite name="PaymentTest" package="com.app.payment" tests="2" failures="1" errors="0" skipped="1" time="2.0">
    <testcase name="testCharge" classname="com.app.payment.PaymentTest" time="1.5">
      <failure message="Amount mismatch" type="AssertionError">expected 50 got 0</failure>
    </testcase>
    <testcase name="testRefund" classname="com.app.payment.PaymentTest" time="0.5">
      <skipped message="not implemented"/>
    </testcase>
  </testsuite>
</testsuites>`)

var singleSuiteXML = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="UnitTests" tests="2" failures="0" errors="0" skipped="0" time="1.0">
  <testcase name="testAdd" classname="MathTest" time="0.3"/>
  <testcase name="testSubtract" classname="MathTest" time="0.7"/>
</testsuite>`)

var malformedXML = []byte(`<not valid xml at all`)

var emptyTestsuitesXML = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="0" failures="0" errors="0" time="0"/>`)

func TestParseMultiSuiteXML(t *testing.T) {
	p := NewJUnitParser()

	report, err := p.Parse(multiSuiteXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalTests != 5 {
		t.Errorf("expected 5 total tests, got %d", report.TotalTests)
	}
	if report.Failed != 2 {
		t.Errorf("expected 2 failures, got %d", report.Failed)
	}
	if report.Errored != 1 {
		t.Errorf("expected 1 error, got %d", report.Errored)
	}
	if report.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", report.Skipped)
	}
	if report.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", report.Passed)
	}
	if len(report.Suites) != 2 {
		t.Errorf("expected 2 suites, got %d", len(report.Suites))
	}
	if report.ID == "" {
		t.Error("expected non-empty report ID")
	}
	if report.Framework != "junit" {
		t.Errorf("expected framework 'junit', got '%s'", report.Framework)
	}
}

func TestParseSingleSuiteXML(t *testing.T) {
	p := NewJUnitParser()

	report, err := p.Parse(singleSuiteXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalTests != 2 {
		t.Errorf("expected 2 total tests, got %d", report.TotalTests)
	}
	if report.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", report.Passed)
	}
	if report.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", report.Failed)
	}
	if len(report.Suites) != 1 {
		t.Errorf("expected 1 suite, got %d", len(report.Suites))
	}
	if report.Suites[0].Name != "UnitTests" {
		t.Errorf("expected suite name 'UnitTests', got '%s'", report.Suites[0].Name)
	}
}

func TestParseMalformedXML(t *testing.T) {
	p := NewJUnitParser()

	_, err := p.Parse(malformedXML)
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}

func TestParseFailureDetails(t *testing.T) {
	p := NewJUnitParser()

	report, err := p.Parse(multiSuiteXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authSuite := report.Suites[0]

	logoutCase := authSuite.Cases[1]
	if logoutCase.Status != core.TestStatusFailed {
		t.Errorf("expected status 'failed', got '%s'", logoutCase.Status)
	}
	if logoutCase.Failure == nil {
		t.Fatal("expected failure details")
	}
	if logoutCase.Failure.Message != "Expected redirect" {
		t.Errorf("expected failure message 'Expected redirect', got '%s'", logoutCase.Failure.Message)
	}
	if logoutCase.Failure.StackTrace != "stack trace here" {
		t.Errorf("expected stack trace 'stack trace here', got '%s'", logoutCase.Failure.StackTrace)
	}

	sessionCase := authSuite.Cases[2]
	if sessionCase.Status != core.TestStatusErrored {
		t.Errorf("expected status 'errored', got '%s'", sessionCase.Status)
	}
	if sessionCase.Failure == nil {
		t.Fatal("expected error details")
	}
	if sessionCase.Failure.Message != "NullPointerException" {
		t.Errorf("expected error message 'NullPointerException', got '%s'", sessionCase.Failure.Message)
	}
}

func TestParseSkippedTests(t *testing.T) {
	p := NewJUnitParser()

	report, err := p.Parse(multiSuiteXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	paymentSuite := report.Suites[1]
	refundCase := paymentSuite.Cases[1]
	if refundCase.Status != core.TestStatusSkipped {
		t.Errorf("expected status 'skipped', got '%s'", refundCase.Status)
	}
}

func TestParseDuration(t *testing.T) {
	p := NewJUnitParser()

	report, err := p.Parse(singleSuiteXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Duration.Seconds() != 1.0 {
		t.Errorf("expected duration 1.0s, got %v", report.Duration)
	}

	addCase := report.Suites[0].Cases[0]
	if addCase.Duration.Seconds() != 0.3 {
		t.Errorf("expected case duration 0.3s, got %v", addCase.Duration)
	}
}

func TestParseSuiteTimestampSetsReportTimestamp(t *testing.T) {
	p := NewJUnitParser()

	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="A" tests="1" timestamp="2026-06-28T10:00:00">
    <testcase name="t1" classname="C" time="0.1"/>
  </testsuite>
  <testsuite name="B" tests="1" timestamp="2026-06-29T11:30:00">
    <testcase name="t2" classname="C" time="0.1"/>
  </testsuite>
</testsuites>`)

	report, err := p.Parse(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := time.Date(2026, 6, 29, 11, 30, 0, 0, time.UTC)
	if !report.Timestamp.Equal(want) {
		t.Errorf("expected report timestamp %v, got %v", want, report.Timestamp)
	}
}

func TestParseSuiteTimestampLayouts(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"2026-06-28T10:00:00", time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)},
		{"2026-06-28T10:00:00Z", time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)},
		{"2026-06-28T10:00:00+02:00", time.Date(2026, 6, 28, 8, 0, 0, 0, time.UTC)},
		{"2026-06-28T10:00:00.123456", time.Date(2026, 6, 28, 10, 0, 0, 123456000, time.UTC)},
		{"2026-06-28 10:00:00", time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		got := parseSuiteTimestamp(tc.input)
		if !got.Equal(tc.want) {
			t.Errorf("parseSuiteTimestamp(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}

	if !parseSuiteTimestamp("").IsZero() {
		t.Error("expected zero time for empty input")
	}
	if !parseSuiteTimestamp("garbage").IsZero() {
		t.Error("expected zero time for unparseable input")
	}
}

func TestParseWithoutSuiteTimestampFallsBackToNow(t *testing.T) {
	p := NewJUnitParser()

	before := time.Now()
	report, err := p.Parse(singleSuiteXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if report.Timestamp.Before(before) || report.Timestamp.After(after) {
		t.Errorf("expected timestamp near now, got %v", report.Timestamp)
	}
}

func TestFormat(t *testing.T) {
	p := NewJUnitParser()
	if p.Format() != "junit" {
		t.Errorf("expected format 'junit', got '%s'", p.Format())
	}
}
