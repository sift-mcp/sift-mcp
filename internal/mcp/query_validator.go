package mcp

import "strings"

type SecurityConfig struct {
	ReadOnly          bool
	AllowRemediation  bool
	MaxQueryResults   int
	MaxQueryTimeRange string
	BlockedPatterns   []string
	AllowedTables     []string
}

func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ReadOnly:          true,
		AllowRemediation:  false,
		MaxQueryResults:   1000,
		MaxQueryTimeRange: "90d",
		BlockedPatterns: []string{
			"DROP",
			"DELETE",
			"UPDATE",
			"INSERT",
			"ALTER",
			"TRUNCATE",
			"CREATE",
			"GRANT",
			"REVOKE",
		},
		AllowedTables: []string{
			"test_reports",
			"test_failures",
		},
	}
}

type SecurityError struct {
	Code    int
	Message string
}

func (e *SecurityError) Error() string {
	return e.Message
}

const (
	ErrCodeBlockedOperation = 1001
	ErrCodeInvalidTable     = 1002
	ErrCodeRateLimited      = 1003
	ErrCodeUnauthorized     = 1004
)

type QueryValidator struct {
	config SecurityConfig
}

func NewQueryValidator(cfg SecurityConfig) *QueryValidator {
	return &QueryValidator{config: cfg}
}

func (v *QueryValidator) ValidateQuery(query string) error {
	upperQuery := strings.ToUpper(query)

	if v.config.ReadOnly {
		for _, pattern := range v.config.BlockedPatterns {
			if strings.Contains(upperQuery, pattern) {
				return &SecurityError{
					Code:    ErrCodeBlockedOperation,
					Message: "operation not allowed in read-only mode: " + pattern,
				}
			}
		}
	}

	return nil
}

func (v *QueryValidator) SanitizeInput(input string) string {
	sanitized := strings.ReplaceAll(input, "'", "''")

	dangerousPatterns := []string{
		"--",
		";",
		"/*",
		"*/",
		"@@",
		"@",
		"char(",
		"exec(",
		"execute(",
		"xp_",
		"sp_",
	}

	for _, pattern := range dangerousPatterns {
		sanitized = strings.ReplaceAll(sanitized, pattern, "")
	}

	return sanitized
}

func (v *QueryValidator) IsAllowedTable(tableName string) bool {
	if len(v.config.AllowedTables) == 0 {
		return true
	}

	lowerTable := strings.ToLower(tableName)
	for _, allowed := range v.config.AllowedTables {
		if strings.ToLower(allowed) == lowerTable {
			return true
		}
	}

	return false
}
