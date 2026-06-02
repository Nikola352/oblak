package sanitizer

import (
	"regexp"
	"strings"
)

type CodeSanitizer struct {
}

func (cs *CodeSanitizer) SanitizeCode(code string) string {
	// 1. Remove Single-line comments (# or //)
	// This kills 99% of "Ignore previous instruction" attempts hidden in comments
	commentRegex := regexp.MustCompile(`(?m)(#|//).*$`)
	code = commentRegex.ReplaceAllString(code, "")

	// 2. Remove Multi-line comments (/* ... */ or """ ... """)
	multiLinePython := regexp.MustCompile(`(?s)"""(.*?)"""|'''(.*?)'''`)
	code = multiLinePython.ReplaceAllString(code, "")

	multiLineGo := regexp.MustCompile(`(?s)/\*.*?\*/`)
	code = multiLineGo.ReplaceAllString(code, "")

	// 3. Trim extra whitespace to save tokens/money
	return strings.TrimSpace(code)
}
