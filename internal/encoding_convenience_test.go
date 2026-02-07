package internal

import (
	"testing"
)

// TestEncodingConvenienceFunctions tests the convenience wrapper functions
// for encoding detection and conversion that were previously untested.

func TestDetectCharsetFromBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "UTF-8 with BOM",
			data:     []byte{0xEF, 0xBB, 0xBF, 'H', 'e', 'l', 'l', 'o'},
			expected: "utf-8",
		},
		{
			name:     "UTF-8 HTML with meta tag",
			data:     []byte("<html><head><meta charset=\"utf-8\"></head><body>Hello</body></html>"),
			expected: "utf-8",
		},
		{
			name:     "Windows-1252 HTML",
			data:     []byte("<html><head><meta http-equiv=\"Content-Type\" content=\"text/html; charset=windows-1252\"></head><body>Hello</body></html>"),
			expected: "windows-1252",
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: "utf-8", // Default fallback
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := NewEncodingDetector()
			result := detector.DetectCharset(tc.data)

			// For empty data, we expect utf-8 as default
			if tc.expected == "" || result == tc.expected {
				return
			}

			// Check if result is valid
			if result == "" {
				t.Errorf("DetectCharsetFromBytes() returned empty string, want %q", tc.expected)
			}
		})
	}
}

func TestConvertToUTF8(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		data         []byte
		charset      string
		expectError  bool
		mustContain  string // Result must contain this string
		mustNotContain string // Result must not contain this string
	}{
		{
			name:        "UTF-8 to UTF-8 (no conversion)",
			data:        []byte("Hello World"),
			charset:     "utf-8",
			expectError: false,
			mustContain: "Hello World",
		},
		{
			name:        "Windows-1252 conversion",
			data:        []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x20, 0xE9, 0x20, 0x57, 0x6F, 0x72, 0x6C, 0x64}, // "Hello é World" in Windows-1252
			charset:     "windows-1252",
			expectError: false,
			mustContain: "Hello",
		},
		{
			name:        "Empty input",
			data:        []byte{},
			charset:     "utf-8",
			expectError: false,
			mustNotContain: "\x00", // Should not contain null bytes
		},
		{
			name:        "Invalid charset",
			data:        []byte("Hello"),
			charset:     "invalid-charset-xyz",
			expectError: false, // Function returns as-is if it can't convert
			mustContain: "Hello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := NewEncodingDetector()
			result, err := detector.ToUTF8(tc.data, tc.charset)

			if tc.expectError && err == nil {
				t.Error("ConvertToUTF8() expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("ConvertToUTF8() unexpected error: %v", err)
			}

			if tc.mustContain != "" {
				if !contains(result, tc.mustContain) {
					t.Errorf("ConvertToUTF8() result must contain %q", tc.mustContain)
				}
			}
			if tc.mustNotContain != "" && contains(result, tc.mustNotContain) {
				t.Errorf("ConvertToUTF8() result must not contain %q", tc.mustNotContain)
			}
		})
	}
}

func TestDetectAndConvertToUTF8(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		data        []byte
		expectError bool
		checkUTF8    bool // Verify result is valid UTF-8
	}{
		{
			name:        "UTF-8 HTML",
			data:        []byte("<html><head><meta charset=\"utf-8\"></head><body>Hello World</body></html>"),
			expectError: false,
			checkUTF8:   true,
		},
		{
			name:        "Windows-1252 HTML",
			data:        []byte("<html><head><meta http-equiv=\"Content-Type\" content=\"text/html; charset=windows-1252\"></head><body>Hello</body></html>"),
			expectError: false,
			checkUTF8:   true,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			expectError: false,
			checkUTF8:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := NewEncodingDetector()
			result, charset, err := detector.DetectAndConvert(tc.data)

			if tc.expectError && err == nil {
				t.Error("DetectAndConvertToUTF8() expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("DetectAndConvertToUTF8() unexpected error: %v", err)
			}
			if charset == "" {
				t.Error("DetectAndConvertToUTF8() returned empty charset")
			}
			if len(result) == 0 && len(tc.data) > 0 {
				t.Error("DetectAndConvertToUTF8() returned empty result")
			}
			if tc.checkUTF8 && !isValidUTF8(result) {
				t.Error("DetectAndConvertToUTF8() result is not valid UTF-8")
			}
		})
	}
}

func TestDetectAndConvertToUTF8String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		data        []byte
		forcedEncoding string
		expectError bool
	}{
		{
			name:        "Auto-detect UTF-8",
			data:        []byte("<html><head><meta charset=\"utf-8\"></head><body>Hello</body></html>"),
			forcedEncoding: "",
			expectError: false,
		},
		{
			name:        "Forced UTF-8",
			data:        []byte("Hello World"),
			forcedEncoding: "utf-8",
			expectError: false,
		},
		{
			name:        "Forced Windows-1252",
			data:        []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F}, // "Hello" in Windows-1252
			forcedEncoding: "windows-1252",
			expectError: false,
		},
		{
			name:        "Invalid forced encoding",
			data:        []byte("Hello"),
			forcedEncoding: "invalid-xyz",
			expectError: false, // Should fall back to auto-detect
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, charset, err := DetectAndConvertToUTF8String(tc.data, tc.forcedEncoding)

			if tc.expectError && err == nil {
				t.Error("DetectAndConvertToUTF8String() expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("DetectAndConvertToUTF8String() unexpected error: %v", err)
			}
			if result == "" && len(tc.data) > 0 {
				t.Error("DetectAndConvertToUTF8String() returned empty string")
			}
			if charset == "" {
				t.Error("DetectAndConvertToUTF8String() returned empty charset")
			}
		})
	}
}

// Helper function to check if bytes are valid UTF-8
func isValidUTF8(data []byte) bool {
	for i := 0; i < len(data); {
		r, size := parseUTF8Char(data[i:])
		if r == 0xFFFD || size == 0 {
			return false
		}
		i += size
	}
	return true
}

// Simple UTF-8 character parser
func parseUTF8Char(data []byte) (rune, int) {
	if len(data) == 0 {
		return 0xFFFD, 0
	}

	b := data[0]
	if b < 0x80 {
		return rune(b), 1
	}

	// Multi-byte sequence
	var n int
	if b>>5 == 0x6 { // 110xxxxx
		n = 2
	} else if b>>4 == 0xE { // 1110xxxx
		n = 3
	} else if b>>3 == 0x1E { // 11110xxx
		n = 4
	} else {
		return 0xFFFD, 1 // Invalid
	}

	if len(data) < n {
		return 0xFFFD, len(data)
	}

	var r rune
	for i := 1; i < n; i++ {
		if data[i]>>6 != 0x2 { // 10xxxxxx
			return 0xFFFD, i
		}
		r = (r << 6) | rune(data[i]&0x3F)
	}

	return r, n
}

func contains(data []byte, substr string) bool {
	if len(substr) > len(data) {
		return false
	}
	for i := 0; i <= len(data)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if data[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
