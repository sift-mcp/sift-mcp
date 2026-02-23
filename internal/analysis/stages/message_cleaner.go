package stages

import (
	"regexp"
	"strings"
	"unicode"

	"sift/internal/core"
)

var noisePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)MergedContextConfiguration@[0-9a-fA-F]+[^\n]*`),
	regexp.MustCompile(`(?i)\btestClass\s*=\s*[^\s,\]\)]+`),
	regexp.MustCompile(`@[0-9a-fA-F]{4,16}\b`),
	regexp.MustCompile(`0x[0-9a-fA-F]+`),
	regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[^\s]*`),
	regexp.MustCompile(`\b(Thread|thread|tid)[- ]?\d+\b`),
	regexp.MustCompile(`\b\d{10,13}\b`),
}

var multipleSpaces = regexp.MustCompile(`\s+`)

func lookupTestFailure(report *core.TestReport, testName string, classname string) *core.TestFailure {
	for _, suite := range report.Suites {
		for _, tc := range suite.Cases {
			if tc.Name == testName && tc.Classname == classname && tc.Failure != nil {
				return tc.Failure
			}
		}
	}
	return nil
}

func stripNoisePatterns(text string) string {
	cleaned := text
	for _, pattern := range noisePatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}
	cleaned = multipleSpaces.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

func smartTruncate(message string, maxLength int) string {
	if len(message) <= maxLength {
		return message
	}

	truncated := message[:maxLength]

	sentenceEnders := []string{". ", "! ", "? "}
	bestBreak := -1
	for _, ender := range sentenceEnders {
		idx := strings.LastIndex(truncated, ender)
		if idx > bestBreak {
			bestBreak = idx + len(ender) - 1
		}
	}

	if bestBreak > maxLength/2 {
		return strings.TrimSpace(truncated[:bestBreak])
	}

	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		return strings.TrimSpace(truncated[:lastSpace])
	}

	lastNonAlnum := -1
	for i := len(truncated) - 1; i > maxLength/2; i-- {
		if !unicode.IsLetter(rune(truncated[i])) && !unicode.IsDigit(rune(truncated[i])) {
			lastNonAlnum = i
			break
		}
	}
	if lastNonAlnum > maxLength/2 {
		return strings.TrimSpace(truncated[:lastNonAlnum])
	}

	return strings.TrimSpace(truncated)
}
