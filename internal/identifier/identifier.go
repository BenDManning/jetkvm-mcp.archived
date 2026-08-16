// Package identifier owns the shared public alias boundary used by
// configuration and MCP tool inputs.
package identifier

import (
	"strings"
	"unicode/utf8"
)

const MaxCodePoints = 128

// Normalize trims an identifier and reports whether it is valid public input.
func Normalize(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	count := utf8.RuneCountInString(trimmed)
	return trimmed, utf8.ValidString(trimmed) && count >= 1 && count <= MaxCodePoints
}
