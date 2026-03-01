package truncate

import "unicode/utf8"

// Truncate shortens text for display, respecting multi-byte characters.
// If the text exceeds maxLen runes, it returns the first maxLen runes with "..." appended.
func Truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}

	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}
