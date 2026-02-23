package mcp

import "regexp"

type PIIScrubber struct {
	patterns map[string]*regexp.Regexp
}

func NewPIIScrubber() *PIIScrubber {
	return &PIIScrubber{
		patterns: map[string]*regexp.Regexp{
			"email":       regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			"ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
			"credit_card": regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
			"phone":       regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
		},
	}
}

func (s *PIIScrubber) Scrub(text string) string {
	result := text

	for piiType, pattern := range s.patterns {
		switch piiType {
		case "email":
			result = pattern.ReplaceAllString(result, "[REDACTED_EMAIL]")
		case "ssn":
			result = pattern.ReplaceAllString(result, "[REDACTED_SSN]")
		case "credit_card":
			result = pattern.ReplaceAllString(result, "[REDACTED_CC]")
		case "phone":
			result = pattern.ReplaceAllString(result, "[REDACTED_PHONE]")
		}
	}

	return result
}

func (s *PIIScrubber) ContainsPII(text string) bool {
	for _, pattern := range s.patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}
