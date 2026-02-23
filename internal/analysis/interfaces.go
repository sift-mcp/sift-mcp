package analysis

import "sift/internal/core"

type FailureClassifier interface {
	Classify(testCase core.TestCase) core.FailureSeverity
	Type() string
}

type FailureAnalyzer interface {
	Analyze(testCase core.TestCase) core.FailedTestInfo
	RegisterClassifier(classifier FailureClassifier)
}
