package stages

import (
	"context"
	"regexp"
	"strings"

	"sift/internal/core"
)

var cascadeIndicatorPatterns = []string{
	"failure threshold",
	"skipping repeated attempt",
}

var testClassExtractPattern = regexp.MustCompile(`(?i)testClass\s*=\s*([^\s,\]\)]+)`)

type CascadeDetectStage struct{}

func NewCascadeDetectStage() *CascadeDetectStage {
	return &CascadeDetectStage{}
}

func (s *CascadeDetectStage) Name() string {
	return "cascade_detect"
}

func (s *CascadeDetectStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	originalsByClassname := make(map[string]*core.FailedTestInfo)
	cascadeIndices := make([]int, 0)

	for i := range result.FailedTests {
		failedTest := &result.FailedTests[i]
		if isCascadeFailure(report, failedTest) {
			cascadeIndices = append(cascadeIndices, i)
		} else {
			if _, exists := originalsByClassname[failedTest.Classname]; !exists {
				originalsByClassname[failedTest.Classname] = failedTest
			}
		}
	}

	for _, idx := range cascadeIndices {
		failedTest := &result.FailedTests[idx]
		failedTest.IsCascade = true
		failedTest.CascadeSourceTest = findCascadeSource(report, failedTest, originalsByClassname)

		if sourceTest := findFailedTestByClassname(result.FailedTests, failedTest.CascadeSourceTest); sourceTest != nil {
			if sourceTest.CleanedRootCause != "" {
				failedTest.CleanedRootCause = sourceTest.CleanedRootCause
			}
		}
	}
}

func isCascadeFailure(report *core.TestReport, failedTest *core.FailedTestInfo) bool {
	failure := lookupTestFailure(report, failedTest.Name, failedTest.Classname)
	if failure == nil {
		return false
	}

	lowerMessage := strings.ToLower(failure.Message)
	for _, indicator := range cascadeIndicatorPatterns {
		if strings.Contains(lowerMessage, indicator) {
			return true
		}
	}

	lowerStackTrace := strings.ToLower(failure.StackTrace)
	for _, indicator := range cascadeIndicatorPatterns {
		if strings.Contains(lowerStackTrace, indicator) {
			return true
		}
	}

	return false
}

func findCascadeSource(report *core.TestReport, cascadeTest *core.FailedTestInfo, originalsByClassname map[string]*core.FailedTestInfo) string {
	if original, exists := originalsByClassname[cascadeTest.Classname]; exists {
		return original.Classname
	}

	failure := lookupTestFailure(report, cascadeTest.Name, cascadeTest.Classname)
	if failure != nil {
		matches := testClassExtractPattern.FindStringSubmatch(failure.Message + " " + failure.StackTrace)
		if len(matches) > 1 {
			extractedClassname := matches[1]
			if original, exists := originalsByClassname[extractedClassname]; exists {
				return original.Classname
			}
		}
	}

	cascadePackage := extractPackagePrefix(cascadeTest.Classname)
	for classname, original := range originalsByClassname {
		if extractPackagePrefix(classname) == cascadePackage {
			return original.Classname
		}
	}

	return ""
}

func extractPackagePrefix(classname string) string {
	lastDot := strings.LastIndex(classname, ".")
	if lastDot >= 0 {
		return classname[:lastDot]
	}
	return classname
}

func findFailedTestByClassname(failedTests []core.FailedTestInfo, classname string) *core.FailedTestInfo {
	if classname == "" {
		return nil
	}
	for i := range failedTests {
		if failedTests[i].Classname == classname && !failedTests[i].IsCascade {
			return &failedTests[i]
		}
	}
	return nil
}
