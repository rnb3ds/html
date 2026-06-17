package html

import "testing"

// TestContainsASCIIFold pins the boundary behavior of the unexported
// containsASCIIFold helper. The function ASCII-case-folds the *haystack* and
// compares against the needle verbatim, so per its sole production caller
// (extract.go, needle "nofollow") the needle must be supplied lower-case. These
// cases exercise that contract at its edges: case folding, no-match, and the
// empty / over-long needle boundaries.
func TestContainsASCIIFold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact lowercase match", "rel=nofollow", "nofollow", true},
		{"mixed-case haystack", "rel=NoFollow", "nofollow", true},
		{"uppercase haystack", "REL=NOFOLLOW", "nofollow", true},
		{"substring embedded", "a nofollow b", "nofollow", true},
		{"no match different word", "rel=dofollow", "nofollow", false},
		{"needle longer than haystack", "nf", "nofollow", false},
		{"empty needle matches anywhere", "anything", "", true},
		{"empty haystack empty needle", "", "", true},
		{"empty haystack non-empty needle", "", "x", false},
		{"ascii prefix before multibyte tail", "caf\xc3\xa9", "caf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsASCIIFold(tt.s, tt.substr); got != tt.want {
				t.Errorf("containsASCIIFold(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
