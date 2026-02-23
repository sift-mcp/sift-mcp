package stages

import (
	"context"
	"regexp"
	"strings"

	"sift/internal/core"
)

var causedByPattern = regexp.MustCompile(`(?i)Caused by:\s*([^\n]+)`)

var assertionKeywords = []string{
	"expected", "but was", "but got", "assertequals", "asserttrue", "assertfalse",
	"assertion failed", "comparison failure",
}

type RootCauseExtractStage struct{}

func NewRootCauseExtractStage() *RootCauseExtractStage {
	return &RootCauseExtractStage{}
}

func (s *RootCauseExtractStage) Name() string {
	return "root_cause_extract"
}

func (s *RootCauseExtractStage) Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult) {
	for i := range result.FailedTests {
		failedTest := &result.FailedTests[i]

		failure := lookupTestFailure(report, failedTest.Name, failedTest.Classname)
		if failure == nil {
			continue
		}

		failedTest.CleanedRootCause = extractCleanRootCause(failure.Message, failure.StackTrace)
	}
}

func extractCleanRootCause(message string, stackTrace string) string {
	combinedText := message + "\n" + stackTrace
	cleanedText := stripNoisePatterns(combinedText)

	causedByChain := extractCausedByChain(cleanedText)
	innermostCause := ""
	if len(causedByChain) > 0 {
		innermostCause = causedByChain[len(causedByChain)-1]
	}

	if result := classifyByDockerTestcontainers(combinedText); result != "" {
		return result
	}

	if result := classifyByDataSource(innermostCause, combinedText); result != "" {
		return result
	}

	if result := classifyByBeanCreation(innermostCause, causedByChain); result != "" {
		return result
	}

	if innermostCause != "" {
		return smartTruncate(innermostCause, 200)
	}

	if result := classifyByAssertion(message); result != "" {
		return result
	}

	cleanedMessage := stripNoisePatterns(message)
	if cleanedMessage != "" {
		return smartTruncate(cleanedMessage, 200)
	}

	return ""
}

func extractCausedByChain(text string) []string {
	matches := causedByPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	chain := make([]string, 0, len(matches))
	for _, match := range matches {
		cause := strings.TrimSpace(match[1])
		if cause != "" {
			chain = append(chain, cause)
		}
	}
	return chain
}

func classifyByDockerTestcontainers(text string) string {
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "testcontainers") || strings.Contains(lowerText, "dockerclientproviderstrategy") || strings.Contains(lowerText, "docker environment") {
		return "Docker daemon not available (required by Testcontainers)"
	}
	if strings.Contains(lowerText, "docker") && (strings.Contains(lowerText, "connection refused") || strings.Contains(lowerText, "not available") || strings.Contains(lowerText, "cannot connect")) {
		return "Docker daemon not available"
	}
	return ""
}

func classifyByDataSource(innermostCause string, fullText string) string {
	lowerFull := strings.ToLower(fullText)
	if !strings.Contains(lowerFull, "datasource") && !strings.Contains(lowerFull, "failed to configure a datasource") {
		return ""
	}

	detail := extractDataSourceDetail(innermostCause, fullText)
	if detail != "" {
		return "DataSource not configured: " + detail
	}
	return "DataSource not configured"
}

func extractDataSourceDetail(innermostCause string, fullText string) string {
	detailPatterns := []string{
		"url", "jdbc", "driver", "password", "username", "connection",
	}
	lowerCause := strings.ToLower(innermostCause)
	for _, pattern := range detailPatterns {
		if strings.Contains(lowerCause, pattern) {
			return smartTruncate(innermostCause, 150)
		}
	}

	lowerFull := strings.ToLower(fullText)
	for _, pattern := range detailPatterns {
		idx := strings.Index(lowerFull, pattern)
		if idx >= 0 {
			endIdx := idx + 150
			if endIdx > len(fullText) {
				endIdx = len(fullText)
			}
			lineEnd := strings.Index(fullText[idx:endIdx], "\n")
			if lineEnd > 0 {
				return smartTruncate(strings.TrimSpace(fullText[idx:idx+lineEnd]), 150)
			}
			return smartTruncate(strings.TrimSpace(fullText[idx:endIdx]), 150)
		}
	}

	return ""
}

func classifyByBeanCreation(innermostCause string, causedByChain []string) string {
	for _, cause := range causedByChain {
		if strings.Contains(strings.ToLower(cause), "beancreationexception") {
			if innermostCause != "" && innermostCause != cause {
				return "Spring bean creation failed: " + smartTruncate(innermostCause, 150)
			}
			return "Spring bean creation failed: " + smartTruncate(cause, 150)
		}
	}
	return ""
}

func classifyByAssertion(message string) string {
	lowerMessage := strings.ToLower(message)
	for _, keyword := range assertionKeywords {
		if strings.Contains(lowerMessage, keyword) {
			cleanedMessage := stripNoisePatterns(message)
			return smartTruncate(cleanedMessage, 200)
		}
	}
	return ""
}
