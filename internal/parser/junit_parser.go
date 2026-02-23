package parser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
	"sift/internal/core"
)

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Package   string          `xml:"package,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      string          `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	Cases     []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure"`
	Error     *junitError   `xml:"error"`
	Skipped   *junitSkipped `xml:"skipped"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

type JUnitParser struct{}

func NewJUnitParser() *JUnitParser {
	return &JUnitParser{}
}

func (p *JUnitParser) Format() string {
	return "junit"
}

func (p *JUnitParser) Parse(data []byte) (*core.TestReport, error) {
	return p.ParseStream(bytes.NewReader(data))
}

func (p *JUnitParser) ParseStream(reader io.Reader) (*core.TestReport, error) {
	report := &core.TestReport{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Source:    "junit",
		Framework: "junit",
		Suites:    make([]core.TestSuite, 0),
	}

	decoder := xml.NewDecoder(reader)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read XML token: %w", err)
		}

		startElem, isStart := token.(xml.StartElement)
		if !isStart {
			continue
		}

		switch startElem.Name.Local {
		case "testsuite":
			var rawSuite junitTestSuite
			if err := decoder.DecodeElement(&rawSuite, &startElem); err != nil {
				return nil, fmt.Errorf("failed to decode testsuite: %w", err)
			}
			suite := p.convertSuite(rawSuite)
			report.Suites = append(report.Suites, suite)
			report.TotalTests += suite.Tests
			report.Failed += suite.Failures
			report.Errored += suite.Errors
			report.Skipped += suite.Skipped
			report.Duration += suite.Duration
		}
	}

	if len(report.Suites) == 0 {
		return nil, fmt.Errorf("unrecognized JUnit XML structure")
	}

	report.Passed = report.TotalTests - report.Failed - report.Errored - report.Skipped

	return report, nil
}

func (p *JUnitParser) convertSuite(raw junitTestSuite) core.TestSuite {
	suite := core.TestSuite{
		Name:     raw.Name,
		Package:  raw.Package,
		Tests:    raw.Tests,
		Failures: raw.Failures,
		Errors:   raw.Errors,
		Skipped:  raw.Skipped,
		Duration: parseDuration(raw.Time),
		Cases:    make([]core.TestCase, 0, len(raw.Cases)),
	}

	if raw.Timestamp != "" {
		if ts, err := time.Parse("2006-01-02T15:04:05", raw.Timestamp); err == nil {
			suite.Timestamp = ts
		}
	}

	for _, rawCase := range raw.Cases {
		suite.Cases = append(suite.Cases, p.convertCase(rawCase))
	}

	return suite
}

func (p *JUnitParser) convertCase(raw junitTestCase) core.TestCase {
	tc := core.TestCase{
		Name:      raw.Name,
		Classname: raw.Classname,
		Duration:  parseDuration(raw.Time),
		Status:    core.TestStatusPassed,
	}

	if raw.Failure != nil {
		tc.Status = core.TestStatusFailed
		tc.Failure = &core.TestFailure{
			Message:    raw.Failure.Message,
			Type:       raw.Failure.Type,
			StackTrace: raw.Failure.Body,
		}
	}

	if raw.Error != nil {
		tc.Status = core.TestStatusErrored
		tc.Failure = &core.TestFailure{
			Message:    raw.Error.Message,
			Type:       raw.Error.Type,
			StackTrace: raw.Error.Body,
		}
	}

	if raw.Skipped != nil {
		tc.Status = core.TestStatusSkipped
	}

	return tc
}

func parseDuration(raw string) time.Duration {
	if raw == "" {
		return 0
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}
