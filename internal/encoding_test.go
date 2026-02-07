package internal

import (
	"bytes"
	"testing"
	"time"
	"unicode/utf8"
)

func TestDetectCharset(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "UTF-8 with BOM",
			data:     []byte{0xEF, 0xBB, 0xBF, '<', 'h', 't', 'm', 'l', '>'},
			expected: "utf-8",
		},
		{
			name:     "UTF-16 LE BOM",
			data:     []byte{0xFF, 0xFE, 0x3C, 0x00},
			expected: "utf-16le",
		},
		{
			name:     "UTF-16 BE BOM",
			data:     []byte{0xFE, 0xFF, 0x00, 0x3C},
			expected: "utf-16be",
		},
		{
			name:     "windows-1252 in meta tag",
			data:     []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=windows-1252"></head></html>`),
			expected: "windows-1252",
		},
		{
			name:     "windows-1252 in meta tag (uppercase)",
			data:     []byte(`<html><head><meta HTTP-EQUIV="Content-Type" CONTENT="text/html; CHARSET=Windows-1252"></head></html>`),
			expected: "windows-1252",
		},
		{
			name:     "iso-8859-1 in meta tag",
			data:     []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1"></head></html>`),
			expected: "iso-8859-1",
		},
		{
			name:     "charset attribute",
			data:     []byte(`<html><head><meta charset="utf-8"></head></html>`),
			expected: "utf-8",
		},
		{
			name:     "shift_jis in meta tag",
			data:     []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=shift_jis"></head></html>`),
			expected: "shift_jis",
		},
		{
			name:     "Valid UTF-8 without charset declaration",
			data:     []byte("<html><head><title>Hello 世界</title></head></html>"),
			expected: "utf-8",
		},
		{
			name:     "Invalid UTF-8 without charset declaration - default to windows-1252",
			data:     []byte{0x92, 0x93, 0x94}, // Invalid UTF-8 bytes
			expected: "windows-1252",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := NewEncodingDetector()
			result := ed.DetectCharset(tt.data)
			if result != tt.expected {
				t.Errorf("DetectCharset() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeCharset(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UTF-8", "utf-8"},
		{"utf8", "utf-8"},
		{"UTF_8", "utf-8"},
		{"WINDOWS-1252", "windows-1252"},
		{"windows1252", "windows-1252"},
		{"CP1252", "windows-1252"},
		{"1252", "windows-1252"},
		{"ISO-8859-1", "iso-8859-1"},
		{"iso88591", "iso-8859-1"},
		{"latin1", "iso-8859-1"},
		{"SHIFT_JIS", "shift_jis"},
		{"shift-jis", "shift_jis"},
		{"sjis", "shift_jis"},
		{"EUC-JP", "euc-jp"},
		{"EUC-KR", "euc-kr"},
		{"GB2312", "gbk"},
		{"BIG5", "big5"},
		{"UTF-16LE", "utf-16le"},
		{"utf16le", "utf-16le"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeCharset(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCharset(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToUTF8_Windows1252(t *testing.T) {
	ed := NewEncodingDetector()

	// Test case: "Registrant's telephone number" with smart quotes
	// In windows-1252: 0x92 = right single quotation mark ' (U+2019)
	input := []byte("Registrant\x92s telephone number, including area code")

	converted, err := ed.ToUTF8(input, "windows-1252")
	if err != nil {
		t.Fatalf("ToUTF8() error = %v", err)
	}

	// 0x92 in windows-1252 converts to U+2019 (RIGHT SINGLE QUOTATION MARK) not U+0027 (APOSTROPHE)
	expected := "Registrant\u2019s telephone number, including area code"
	if string(converted) != expected {
		t.Errorf("ToUTF8(windows-1252) = %q, want %q", string(converted), expected)
	}
}

func TestToUTF8_CommonWindows1252Chars(t *testing.T) {
	ed := NewEncodingDetector()

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Right single quote",
			input:    []byte{0x92},
			expected: "\u2019", // U+2019 RIGHT SINGLE QUOTATION MARK
		},
		{
			name:     "Left double quote",
			input:    []byte{0x93},
			expected: "\u201C", // Left double quotation mark
		},
		{
			name:     "Right double quote",
			input:    []byte{0x94},
			expected: "\u201D", // Right double quotation mark
		},
		{
			name:     "En dash",
			input:    []byte{0x96},
			expected: "–",
		},
		{
			name:     "Em dash",
			input:    []byte{0x97},
			expected: "—",
		},
		{
			name:     "Bullet",
			input:    []byte{0x95},
			expected: "•",
		},
		{
			name:     "Euro sign",
			input:    []byte{0x80},
			expected: "€",
		},
		{
			name:     "Trademark",
			input:    []byte{0x99},
			expected: "™",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted, err := ed.ToUTF8(tt.input, "windows-1252")
			if err != nil {
				t.Fatalf("ToUTF8() error = %v", err)
			}

			if string(converted) != tt.expected {
				t.Errorf("ToUTF8(windows-1252, byte 0x%02x) = %q (U+%04X), want %q (U+%04X)",
					tt.input[0],
					string(converted),
					[]rune(string(converted))[0],
					tt.expected,
					[]rune(tt.expected)[0])
			}
		})
	}
}

func TestToUTF8_UTF8(t *testing.T) {
	ed := NewEncodingDetector()

	input := []byte("Hello 世界 🌍")
	converted, err := ed.ToUTF8(input, "utf-8")

	if err != nil {
		t.Fatalf("ToUTF8() error = %v", err)
	}

	if !bytes.Equal(input, converted) {
		t.Errorf("ToUTF8(utf-8) should return unchanged input, got %q", string(converted))
	}
}

func TestToUTF8_ISO8859_1(t *testing.T) {
	ed := NewEncodingDetector()

	// Test ISO-8859-1: é is 0xE9 in ISO-8859-1, which encodes to U+00E9 in UTF-8
	input := []byte{0x43, 0x61, 0x66, 0xE9} // "Café" in ISO-8859-1
	converted, err := ed.ToUTF8(input, "iso-8859-1")

	if err != nil {
		t.Fatalf("ToUTF8() error = %v", err)
	}

	expected := "Caf\u00E9" // "Café" in UTF-8
	if string(converted) != expected {
		t.Errorf("ToUTF8(iso-8859-1) = %q, want %q", string(converted), expected)
	}
}

func TestDetectAndConvert(t *testing.T) {
	tests := []struct {
		name            string
		data            []byte
		expectedCharset string
		expectedText    string
	}{
		{
			name:            "UTF-8 HTML",
			data:            []byte(`<html><body>Hello World</body></html>`),
			expectedCharset: "utf-8",
			expectedText:    `<html><body>Hello World</body></html>`,
		},
		{
			name:            "windows-1252 HTML with smart quote",
			data:            []byte("<html><head><meta charset=\"windows-1252\"></head><body>Registrant\x92s test</body></html>"),
			expectedCharset: "windows-1252",
			expectedText:    "<html><head><meta charset=\"windows-1252\"></head><body>Registrant\u2019s test</body></html>",
		},
		{
			name:            "Valid UTF-8 without charset",
			data:            []byte(`<html><body>Hello 世界</body></html>`),
			expectedCharset: "utf-8",
			expectedText:    `<html><body>Hello 世界</body></html>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := NewEncodingDetector()
			converted, charset, err := ed.DetectAndConvert(tt.data)

			if err != nil {
				t.Fatalf("DetectAndConvert() error = %v", err)
			}

			if charset != tt.expectedCharset {
				t.Errorf("DetectAndConvert() charset = %v, want %v", charset, tt.expectedCharset)
			}

			if string(converted) != tt.expectedText {
				t.Errorf("DetectAndConvert() text = %q, want %q", string(converted), tt.expectedText)
			}
		})
	}
}

func TestForcedEncoding(t *testing.T) {
	ed := NewEncodingDetector()
	ed.ForcedEncoding = "windows-1252"

	// Even though this is valid UTF-8, forcing windows-1252 should interpret it as such
	input := []byte("Test\x92\x93\x94")
	converted, charset, err := ed.DetectAndConvert(input)

	if err != nil {
		t.Fatalf("DetectAndConvert() error = %v", err)
	}

	if charset != "windows-1252" {
		t.Errorf("Expected charset windows-1252, got %v", charset)
	}

	expected := "Test\u2019\u201C\u201D" // 0x92->', 0x93->", 0x94->"
	if string(converted) != expected {
		t.Errorf("Forced encoding conversion = %q, want %q", string(converted), expected)
	}
}

func BenchmarkDetectCharset(b *testing.B) {
	data := []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=windows-1252"><title>Test</title></head><body>Content</body></html>`)

	ed := NewEncodingDetector()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ed.DetectCharset(data)
	}
}

func BenchmarkToUTF8_Windows1252(b *testing.B) {
	// 1KB of data
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	ed := NewEncodingDetector()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ed.ToUTF8(data, "windows-1252")
	}
}

func BenchmarkDetectAndConvert(b *testing.B) {
	data := []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=windows-1252"></head><body>Test\x92\x93\x94\x95\x96\x97\x99\x80</body></html>`)

	ed := NewEncodingDetector()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ed.DetectAndConvert(data)
	}
}

// TestSmartEncodingDetection tests the intelligent encoding detection system
func TestSmartEncodingDetection(t *testing.T) {
	tests := []struct {
		name            string
		data            []byte
		expectedCharset string
		minConfidence   int
		description     string
	}{
		{
			name:            "UTF-8 with Chinese",
			data:            []byte(`<html><head><meta charset="utf-8"></head><body>你好世界</body></html>`),
			expectedCharset: "utf-8",
			minConfidence:   80,
			description:     "UTF-8 with CJK characters",
		},
		{
			name:            "GBK meta tag but UTF-8 data (common bug)",
			data:            createGBKEncodedHTML(),
			expectedCharset: "utf-8", // Smart detection should detect actual UTF-8
			minConfidence:   70,
			description:     "Meta tag declares GBK but data is actually UTF-8",
		},
		{
			name:            "Windows-1252 with smart quotes",
			data:            createWindows1252HTML(),
			expectedCharset: "windows-1252",
			minConfidence:   60,
			description:     "Western text with Windows-1252 special chars",
		},
		{
			name:            "UTF-8 with Cyrillic",
			data:            []byte(`<html><head><meta charset="utf-8"></head><body>Привет мир</body></html>`),
			expectedCharset: "utf-8",
			minConfidence:   80,
			description:     "UTF-8 with Cyrillic characters",
		},
		{
			name:            "ASCII-only defaults to UTF-8",
			data:            []byte(`<html><body>Hello World</body></html>`),
			expectedCharset: "utf-8",
			minConfidence:   50,
			description:     "ASCII-only content defaults to UTF-8",
		},
		{
			name:            "UTF-8 with accented chars",
			data:            []byte(`<html><body>Café résumé naïve</body></html>`),
			expectedCharset: "utf-8",
			minConfidence:   80,
			description:     "UTF-8 with accented Latin characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := NewEncodingDetector()
			ed.EnableSmartDetection = true

			// Test smart detection
			match := ed.DetectCharsetSmart(tt.data)

			t.Logf("Test: %s", tt.description)
			t.Logf("  Detected: %s (confidence: %d, score: %d)", match.Charset, match.Confidence, match.Score)

			if match.Charset != tt.expectedCharset {
				t.Errorf("Expected charset %s, got %s", tt.expectedCharset, match.Charset)
			}

			if match.Confidence < tt.minConfidence {
				t.Errorf("Confidence too low: %d (expected >= %d)", match.Confidence, tt.minConfidence)
			}

			// Verify conversion works
			converted, charset, err := ed.DetectAndConvert(tt.data)
			if err != nil {
				t.Errorf("DetectAndConvert failed: %v", err)
			}

			if charset != tt.expectedCharset {
				t.Errorf("DetectAndConvert charset mismatch: %s != %s", charset, tt.expectedCharset)
			}

			// Verify result is valid UTF-8
			if !utf8.Valid(converted) {
				t.Errorf("Converted data is not valid UTF-8")
			}
		})
	}
}

// TestSmartDetectionPerformance benchmarks the smart detection system
func TestSmartDetectionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create test data of different sizes
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10240},
		{"100KB", 102400},
	}

	encodings := []struct {
		name string
		data []byte
	}{
		{"UTF-8", []byte(`<html><head><meta charset="utf-8"></head><body>Café résumé 你好世界</body></html>`)},
		{"Windows-1252", createWindows1252HTML()},
	}

	for _, sizeInfo := range sizes {
		t.Run(sizeInfo.name, func(t *testing.T) {
			for _, encInfo := range encodings {
				t.Run(encInfo.name, func(t *testing.T) {
					// Create data of specified size
					data := make([]byte, 0, sizeInfo.size)
					for len(data) < sizeInfo.size {
						data = append(data, encInfo.data...)
					}
					data = data[:sizeInfo.size]

					ed := NewEncodingDetector()
					ed.EnableSmartDetection = true

					// Measure detection time
					start := time.Now()
					match := ed.DetectCharsetSmart(data)
					duration := time.Since(start)

					t.Logf("Size: %s, Encoding: %s, Detected: %s (confidence: %d), Time: %v",
						sizeInfo.name, encInfo.name, match.Charset, match.Confidence, duration)

					// Detection should complete in reasonable time (< 1 second for 100KB)
					if duration > time.Second {
						t.Errorf("Detection took too long: %v", duration)
					}
				})
			}
		})
	}
}

// Helper functions to create test data

func createWindows1252HTML() []byte {
	base := []byte(`<html><head><meta charset="windows-1252"></head><body>`)
	// Add text with Windows-1252 special characters (byte 0x92 = ')
	text := []byte("Registrant\x92s telephone number")
	suffix := []byte(`</body></html>`)
	result := make([]byte, len(base)+len(text)+len(suffix))
	copy(result, base)
	copy(result[len(base):], text)
	copy(result[len(base)+len(text):], suffix)
	return result
}

func createGBKEncodedHTML() []byte {
	// Return a HTML with Chinese characters
	// In actual GBK encoding this would be different bytes,
	// but for testing we use UTF-8 which should be detected
	return []byte(`<html><head><meta charset="gbk"></head><body>你好世界</body></html>`)
}

// Test for the specific bug: UTF-8 file declaring Windows-1252 should be detected as UTF-8
func TestUTF8WithWrongCharsetMeta(t *testing.T) {
	tests := []struct {
		name            string
		data            []byte
		expectedCharset string
		shouldContain   string
	}{
		{
			name: "UTF-8 with accented chars and Windows-1252 meta",
			data: []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=windows-1252"></head><body>Café résumé</body></html>`),
			// café has UTF-8 bytes: 63 61 66 C3 A9 20
			// This should be detected as UTF-8 because it has valid UTF-8 multi-byte sequences
			expectedCharset: "utf-8",
			shouldContain:   "Café",
		},
		{
			name: "UTF-8 with Chinese chars and Windows-1252 meta",
			data: []byte(`<html><head><meta charset="windows-1252"></head><body>Hello 世界</body></html>`),
			expectedCharset: "utf-8",
			shouldContain:   "世界",
		},
		{
			name: "Proper Windows-1252 with 0x92 apostrophe",
			// Create the byte array manually to ensure 0x92 is a raw byte, not escape sequence
			data: func() []byte {
				base := []byte(`<html><head><meta charset="windows-1252"></head><body>Registrant`)
				suffix := []byte(`s</body></html>`)
				result := make([]byte, len(base)+1+len(suffix))
				copy(result, base)
				result[len(base)] = 0x92
				copy(result[len(base)+1:], suffix)
				return result
			}(),
			// This is actual Windows-1252 (byte 0x92 is not valid UTF-8)
			expectedCharset: "windows-1252",
			shouldContain:   "\u2019", // 0x92 in Windows-1252 is U+2019 RIGHT SINGLE QUOTATION MARK
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := NewEncodingDetector()
			charset := ed.DetectCharset(tt.data)

			if charset != tt.expectedCharset {
				t.Errorf("DetectCharset() = %v, want %v", charset, tt.expectedCharset)
			}

			// Also verify conversion works
			converted, _, err := ed.DetectAndConvert(tt.data)
			if err != nil {
				t.Fatalf("DetectAndConvert() error = %v", err)
			}

			result := string(converted)
			if !bytes.Contains(converted, []byte(tt.shouldContain)) {
				t.Errorf("Expected output to contain %q, got: %s", tt.shouldContain, result)
			}
		})
	}
}

