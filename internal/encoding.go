// Package internal provides character encoding detection and conversion functionality.
// It supports 15+ encodings including Unicode variants, Western European,
// and East Asian character sets, with intelligent auto-detection capabilities.
package internal

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	// Pre-compiled regex patterns for charset detection
	charsetPattern    = regexp.MustCompile(`(?i)<meta\s+[^>]*http-equiv=["']?content-type["']?[^>]*content=["']?[^;]*;\s*charset=([^"'\s>]+)`)
	charsetPatternAlt = regexp.MustCompile(`(?i)<meta\s+charset=["']?([^"'\s>]+)`)
)

// EncodingDetector handles charset detection and conversion
type EncodingDetector struct {
	// User-specified encoding override (optional)
	ForcedEncoding string

	// Smart detection options
	EnableSmartDetection bool // Enable intelligent encoding detection
	MaxSampleSize        int  // Max bytes to analyze for statistical detection
}

// NewEncodingDetector creates a new encoding detector with smart detection enabled
func NewEncodingDetector() *EncodingDetector {
	return &EncodingDetector{
		EnableSmartDetection: true,
		MaxSampleSize:        10240, // Analyze first 10KB
	}
}

// EncodingMatch represents a detected encoding with confidence score
type EncodingMatch struct {
	Charset    string
	Confidence int  // 0-100
	Score      int  // Detailed score
	Valid      bool // Whether decoding produced valid UTF-8
}

// DetectCharset attempts to detect the character encoding from HTML content
func (ed *EncodingDetector) DetectCharset(data []byte) string {
	// If user forced a specific encoding, use it
	if ed.ForcedEncoding != "" {
		return normalizeCharset(ed.ForcedEncoding)
	}

	// Use smart detection if enabled
	if ed.EnableSmartDetection {
		if match := ed.DetectCharsetSmart(data); match.Confidence >= 80 {
			return match.Charset
		}
	}

	return ed.DetectCharsetBasic(data)
}

// DetectCharsetBasic performs basic charset detection (BOM, meta tags, UTF-8 validation)
func (ed *EncodingDetector) DetectCharsetBasic(data []byte) string {
	// Check for UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "utf-8"
	}

	// Check for UTF-16 BE BOM
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return "utf-16be"
	}

	// Check for UTF-16 LE BOM
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return "utf-16le"
	}

	// IMPORTANT: Check if data is valid UTF-8 BEFORE trusting meta tags.
	// Many HTML files incorrectly declare charset in meta tags while actually being UTF-8.
	// If the data is valid UTF-8, we should trust it over the meta tag declaration.
	if utf8.Valid(data) {
		// Data appears to be valid UTF-8, but let's check the meta tag anyway
		// to avoid false positives (e.g., ASCII-only Windows-1252 files are also valid UTF-8)
		sampleSize := 1024
		if len(data) > sampleSize {
			data = data[:sampleSize]
		}
		htmlStart := string(data)

		// If meta tag explicitly declares UTF-8, confirm it
		if matches := charsetPattern.FindStringSubmatch(htmlStart); len(matches) > 1 {
			declaredCharset := normalizeCharset(matches[1])
			if declaredCharset == "utf-8" {
				return "utf-8"
			}
		}
		if matches := charsetPatternAlt.FindStringSubmatch(htmlStart); len(matches) > 1 {
			declaredCharset := normalizeCharset(matches[1])
			if declaredCharset == "utf-8" {
				return "utf-8"
			}
		}

		// If data is valid UTF-8 with non-ASCII characters, trust the data over meta tag
		// (meta tag is likely wrong). Only do this for files with actual UTF-8 sequences.
		if hasUTF8Sequences(data) {
			return "utf-8"
		}
	}

	// Try to detect from meta tags (first 1024 bytes should be enough)
	sampleSize := 1024
	if len(data) > sampleSize {
		data = data[:sampleSize]
	}
	htmlStart := string(data)

	// Try primary pattern: <meta http-equiv="Content-Type" content="... charset=...">
	if matches := charsetPattern.FindStringSubmatch(htmlStart); len(matches) > 1 {
		return normalizeCharset(matches[1])
	}

	// Try alternative pattern: <meta charset="...">
	if matches := charsetPatternAlt.FindStringSubmatch(htmlStart); len(matches) > 1 {
		return normalizeCharset(matches[1])
	}

	// If no charset declared and data appears to be valid UTF-8, assume UTF-8
	if utf8.Valid(data) {
		return "utf-8"
	}

	// Default to windows-1252 for HTML (most common fallback for Western content)
	return "windows-1252"
}

// DetectCharsetSmart performs intelligent charset detection using statistical analysis
func (ed *EncodingDetector) DetectCharsetSmart(data []byte) EncodingMatch {
	// First, try basic detection
	basicCharset := ed.DetectCharsetBasic(data)

	// Step 1: Quick validation of basic detection
	// For meta-tag declared UTF-8, give it high priority
	score := ed.scoreEncodingMatch(data, basicCharset)

	// Check if basic detection came from meta tag
	if basicCharset == "utf-8" && score >= 70 {
		// Meta tag declared UTF-8 with good score - trust it
		return EncodingMatch{
			Charset:    basicCharset,
			Confidence: 90,
			Score:      score,
			Valid:      true,
		}
	}

	if score >= 90 {
		return EncodingMatch{
			Charset:    basicCharset,
			Confidence: 95,
			Score:      score,
			Valid:      true,
		}
	}

	// Step 2: Try multiple encodings and pick the best match
	matches := ed.tryAllEncodings(data)

	// Boost score for basicCharset if it's from meta tag
	for i := range matches {
		if matches[i].Charset == basicCharset {
			matches[i].Score += 10
			matches[i].Confidence += 5
			break
		}
	}

	// Step 3: Validate and score each encoding
	var bestMatch EncodingMatch
	for _, match := range matches {
		if match.Score > bestMatch.Score || (match.Score == bestMatch.Score && match.Confidence > bestMatch.Confidence) {
			bestMatch = match
		}
	}

	// If all matches have low scores, fall back to basic detection
	if bestMatch.Confidence < 50 {
		bestMatch = EncodingMatch{
			Charset:    basicCharset,
			Confidence: 50,
			Score:      score,
			Valid:      utf8.Valid(data),
		}
	}

	return bestMatch
}

// hasUTF8Sequences checks if data contains actual UTF-8 multi-byte sequences
// (not just ASCII which is also valid UTF-8)
func hasUTF8Sequences(data []byte) bool {
	for i := 0; i < len(data); i++ {
		// Check for UTF-8 multi-byte sequences
		if data[i] >= 0x80 {
			// Found a byte that's >= 0x80, which in UTF-8 starts a multi-byte sequence
			// If this is valid UTF-8, we have real UTF-8 content
			return true
		}
	}
	return false
}

// ToUTF8 converts the given data from the detected charset to UTF-8
func (ed *EncodingDetector) ToUTF8(data []byte, charset string) ([]byte, error) {
	charset = normalizeCharset(charset)

	// If already UTF-8, return as-is
	if charset == "utf-8" || charset == "utf8" {
		return data, nil
	}

	// Get the appropriate encoding
	enc := getEncoding(charset)
	if enc == nil {
		// Unknown encoding, try to return as-is if valid UTF-8
		if utf8.Valid(data) {
			return data, nil
		}
		// Otherwise, return with a note that encoding couldn't be determined
		return data, nil
	}

	// Create a transformer
	transformer := enc.NewDecoder()

	// Convert the data
	reader := transform.NewReader(bytes.NewReader(data), transformer)
	converted, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return converted, nil
}

// DetectAndConvert detects charset and converts to UTF-8 in one step
func (ed *EncodingDetector) DetectAndConvert(data []byte) ([]byte, string, error) {
	var charset string
	if ed.EnableSmartDetection {
		match := ed.DetectCharsetSmart(data)
		charset = match.Charset
	} else {
		charset = ed.DetectCharset(data)
	}
	converted, err := ed.ToUTF8(data, charset)
	return converted, charset, err
}

// normalizeCharset normalizes charset names to a standard form
func normalizeCharset(charset string) string {
	charset = strings.ToLower(strings.TrimSpace(charset))

	// Remove common prefixes and suffixes
	charset = strings.TrimPrefix(charset, "text/")
	charset = strings.TrimPrefix(charset, "text-")
	charset = strings.TrimPrefix(charset, "windows-")
	charset = strings.TrimPrefix(charset, "cp")
	charset = strings.TrimPrefix(charset, "codepage-")
	charset = strings.TrimPrefix(charset, "ibm-")
	charset = strings.TrimPrefix(charset, "iso-")
	charset = strings.TrimPrefix(charset, "iso_")
	// Don't use TrimPrefix for "latin" as it would match "latin1" -> "1"
	if strings.HasPrefix(charset, "latin") && len(charset) > 5 {
		charset = "iso-8859-1" // latin1, latin-1, etc. defaults to iso-8859-1
	}

	// Handle specific mappings
	switch charset {
	case "1252", "cp1252", "windows1252":
		return "windows-1252"
	case "1251", "cp1251", "windows1251":
		return "windows-1251"
	case "1250", "cp1250", "windows1250":
		return "windows-1250"
	case "8859-1", "88591", "iso88591", "iso_8859-1", "iso_8859_1", "latin1", "latin-1":
		return "iso-8859-1"
	case "8859-15", "885915", "iso885915", "iso_8859-15", "iso_8859_15":
		return "iso-8859-15"
	case "utf8", "utf-8", "utf_8":
		return "utf-8"
	case "utf16", "utf-16", "utf_16", "utf16le", "utf-16le":
		return "utf-16le"
	case "utf16be", "utf-16be":
		return "utf-16be"
	case "shift_jis", "shift-jis", "shiftjis", "sjis", "x-sjis":
		return "shift_jis"
	case "euc-jp", "euc_jp", "eucjp":
		return "euc-jp"
	case "euc-kr", "euc_kr", "euckr":
		return "euc-kr"
	case "gb2312", "gb2312-80", "gb2312_80":
		return "gbk"
	case "gbk":
		return "gbk"
	case "big5", "big-5", "big5-hkscs":
		return "big5"
	}

	return charset
}

// getEncoding returns the encoding for the given charset name
func getEncoding(charset string) encoding.Encoding {
	switch charset {
	case "windows-1252":
		return charmap.Windows1252
	case "windows-1251":
		return charmap.Windows1251 // Cyrillic
	case "windows-1250":
		return charmap.Windows1250 // Central European
	case "iso-8859-1":
		return charmap.ISO8859_1
	case "iso-8859-15":
		return charmap.ISO8859_15 // Western European with Euro
	case "iso-8859-2":
		return charmap.ISO8859_2 // Central European
	case "iso-8859-3":
		return charmap.ISO8859_3 // South European
	case "iso-8859-4":
		return charmap.ISO8859_4 // Baltic
	case "iso-8859-5":
		return charmap.ISO8859_5 // Cyrillic
	case "iso-8859-6":
		return charmap.ISO8859_6 // Arabic
	case "iso-8859-7":
		return charmap.ISO8859_7 // Greek
	case "iso-8859-8":
		return charmap.ISO8859_8 // Hebrew
	case "iso-8859-9":
		return charmap.ISO8859_9 // Turkish
	case "iso-8859-10":
		return charmap.ISO8859_10 // Nordic
	case "iso-8859-13":
		return charmap.ISO8859_13 // Baltic
	case "iso-8859-14":
		return charmap.ISO8859_14 // Celtic
	case "iso-8859-16":
		return charmap.ISO8859_16 // Southeast European
	case "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	case "shift_jis":
		return japanese.ShiftJIS
	case "euc-jp":
		return japanese.EUCJP
	case "iso-2022-jp":
		return japanese.ISO2022JP
	case "euc-kr":
		return korean.EUCKR
	case "gbk":
		return simplifiedchinese.GBK
	case "big5":
		return traditionalchinese.Big5
	default:
		return nil
	}
}

// DetectCharsetFromBytes is a convenience function that detects charset from byte data
func DetectCharsetFromBytes(data []byte) string {
	ed := NewEncodingDetector()
	return ed.DetectCharset(data)
}

// ConvertToUTF8 is a convenience function that converts data to UTF-8
func ConvertToUTF8(data []byte, charset string) ([]byte, error) {
	ed := NewEncodingDetector()
	return ed.ToUTF8(data, charset)
}

// DetectAndConvertToUTF8 is a convenience function that detects charset and converts to UTF-8
func DetectAndConvertToUTF8(data []byte) ([]byte, string, error) {
	ed := NewEncodingDetector()
	return ed.DetectAndConvert(data)
}

// DetectAndConvertToUTF8String detects encoding and converts to UTF-8 string.
// If forcedEncoding is not empty, it will use that encoding instead of auto-detection.
// Returns a UTF-8 string and the detected/used encoding.
func DetectAndConvertToUTF8String(data []byte, forcedEncoding string) (string, string, error) {
	ed := NewEncodingDetector()
	if forcedEncoding != "" {
		ed.ForcedEncoding = forcedEncoding
	}

	convertedBytes, detectedCharset, err := ed.DetectAndConvert(data)
	if err != nil {
		return "", "", err
	}

	return string(convertedBytes), detectedCharset, nil
}

// tryAllEncodings attempts to decode the data with multiple encodings and scores each result.
// Optimized to avoid redundant UTF-8 validation and conversions.
func (ed *EncodingDetector) tryAllEncodings(data []byte) []EncodingMatch {
	// Common encodings to try, ordered by likelihood
	candidateEncodings := []struct {
		name string
		prio int // Priority (higher = more likely)
	}{
		{"utf-8", 100},
		{"windows-1252", 90},
		{"gbk", 80},          // Simplified Chinese
		{"shift_jis", 75},    // Japanese
		{"euc-jp", 70},       // Japanese
		{"euc-kr", 65},       // Korean
		{"big5", 60},         // Traditional Chinese
		{"iso-8859-1", 50},   // Western European
		{"iso-8859-2", 45},   // Central European
		{"windows-1250", 43}, // Central European
		{"windows-1251", 40}, // Cyrillic
		{"iso-8859-5", 38},   // Cyrillic
		{"iso-2022-jp", 35},  // Japanese (ISO-2022-JP)
	}

	// Pre-check UTF-8 validity once (avoids redundant checks in scoreEncodingMatch)
	isUTF8Valid := utf8.Valid(data)

	matches := make([]EncodingMatch, 0, len(candidateEncodings))

	for _, candidate := range candidateEncodings {
		score := ed.scoreEncodingMatchOptimized(data, candidate.name, isUTF8Valid)
		if score > 0 {
			confidence := calculateConfidence(score, candidate.prio)
			matches = append(matches, EncodingMatch{
				Charset:    candidate.name,
				Confidence: confidence,
				Score:      score,
				Valid:      score >= 40, // Threshold for considering it valid
			})
		}
	}

	return matches
}

// scoreEncodingMatchOptimized scores how well a charset matches the data.
// Optimized version that accepts pre-computed UTF-8 validity to avoid redundant checks.
func (ed *EncodingDetector) scoreEncodingMatchOptimized(data []byte, charset string, isUTF8Valid bool) int {
	normalizedCharset := normalizeCharset(charset)

	// UTF-8 special case: use pre-computed validity
	if normalizedCharset == "utf-8" {
		if !isUTF8Valid {
			return 0
		}
		return ed.scoreUTF8(data)
	}

	// Get the encoding
	enc := getEncoding(normalizedCharset)
	if enc == nil {
		return 0
	}

	// Attempt decoding
	transformer := enc.NewDecoder()
	reader := transform.NewReader(bytes.NewReader(data), transformer)
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return 0
	}

	// Score the decoded result
	return ed.scoreDecodedData(decoded, data, normalizedCharset)
}

// scoreEncodingMatch scores how well a charset matches the data
func (ed *EncodingDetector) scoreEncodingMatch(data []byte, charset string) int {
	return ed.scoreEncodingMatchOptimized(data, charset, utf8.Valid(data))
}

// scoreUTF8 scores UTF-8 data
func (ed *EncodingDetector) scoreUTF8(data []byte) int {
	if !utf8.Valid(data) {
		return 0
	}

	score := 0

	// Base score for valid UTF-8
	score += 40

	// Bonus for non-ASCII content (real UTF-8, not just ASCII)
	if hasUTF8Sequences(data) {
		score += 30
	}

	// Calculate printable character ratio
	printableRatio := calculatePrintableRatio(data)
	score += int(printableRatio * 20)

	// Check for proper UTF-8 structure
	validUTF8Ratio := calculateValidUTF8Ratio(data)
	score += int(validUTF8Ratio * 10)

	return score
}

// scoreDecodedData scores decoded data from non-UTF-8 encodings
func (ed *EncodingDetector) scoreDecodedData(decoded, original []byte, charset string) int {
	score := 0

	// Base score for successful decoding
	score += 40

	// Check if result is valid UTF-8
	if !utf8.Valid(decoded) {
		return score - 50 // Heavy penalty for invalid UTF-8 output
	}
	score += 30

	// Calculate printable character ratio
	printableRatio := calculatePrintableRatio(decoded)
	if printableRatio < 0.5 {
		return score - 30 // Heavy penalty for low printable ratio
	}
	score += int(printableRatio * 20)

	// Check for replacement characters (U+FFFD)
	if bytes.Contains(decoded, []byte{0xEF, 0xBF, 0xBD}) {
		score -= 15 // Penalty for replacement characters
	}

	// Bonus for language-specific patterns
	score += ed.scoreLanguagePatterns(decoded, charset)

	// Penalty for excessive control characters
	if hasExcessiveControlChars(decoded) {
		score -= 10
	}

	return score
}

// scoreLanguagePatterns scores based on language-specific character patterns
func (ed *EncodingDetector) scoreLanguagePatterns(decoded []byte, charset string) int {
	bonus := 0

	// Count CJK characters (Chinese, Japanese, Korean)
	cjkCount := countCJKCharacters(decoded)
	if cjkCount > 0 {
		// Expected CJK for certain charsets
		switch charset {
		case "gbk", "big5", "shift_jis", "euc-jp", "euc-kr", "iso-2022-jp":
			// These charsets should have CJK characters
			cjkRatio := float64(cjkCount) / float64(len(decoded))
			bonus += int(cjkRatio * 15)
		default:
			// Western charsets with CJK: penalty
			bonus -= 10
		}
	}

	// Check for Cyrillic characters
	if hasCyrillicCharacters(decoded) {
		if charset == "windows-1251" || charset == "iso-8859-5" {
			bonus += 10
		}
	}

	return bonus
}

// calculateConfidence calculates the final confidence score based on match score and priority
func calculateConfidence(score, priority int) int {
	// Base confidence from score
	confidence := score

	// Adjust by priority
	if priority >= 90 {
		confidence += 5
	} else if priority >= 70 {
		confidence += 2
	}

	// Ensure confidence is in range [0, 100]
	if confidence > 100 {
		confidence = 100
	}
	if confidence < 0 {
		confidence = 0
	}

	return confidence
}

// Helper functions for character analysis

// calculatePrintableRatio calculates the ratio of printable characters
func calculatePrintableRatio(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	printable := 0
	for _, b := range data {
		if isPrintable(b) {
			printable++
		}
	}
	return float64(printable) / float64(len(data))
}

// isPrintable checks if a byte is a printable character
func isPrintable(b byte) bool {
	// ASCII printable range
	if b >= 32 && b <= 126 {
		return true
	}
	// Common whitespace
	if b == '\t' || b == '\n' || b == '\r' {
		return true
	}
	// Non-ASCII UTF-8 continuation bytes are likely valid
	if b >= 0x80 {
		return true
	}
	return false
}

// calculateValidUTF8Ratio calculates the ratio of valid UTF-8 sequences
func calculateValidUTF8Ratio(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	valid := 0
	i := 0
	for i < len(data) {
		r, size := utf8.DecodeRune(data[i:])
		if r != utf8.RuneError {
			valid++
		}
		i += size
		if size == 0 {
			break
		}
	}
	return float64(valid) / float64(len(data))
}

// countCJKCharacters counts CJK (Chinese, Japanese, Korean) characters
func countCJKCharacters(data []byte) int {
	count := 0
	for _, r := range string(data) {
		// CJK Unified Ideographs block
		if (r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0x3400 && r <= 0x4DBF) ||
			(r >= 0x20000 && r <= 0x2A6DF) ||
			(r >= 0x2A700 && r <= 0x2B73F) ||
			(r >= 0x2B740 && r <= 0x2B81F) ||
			(r >= 0x2B820 && r <= 0x2CEAF) ||
			(r >= 0x2CEB0 && r <= 0x2EBEF) ||
			// Hiragana and Katakana
			(r >= 0x3040 && r <= 0x309F) ||
			(r >= 0x30A0 && r <= 0x30FF) ||
			// Hangul Syllables
			(r >= 0xAC00 && r <= 0xD7AF) ||
			// CJK Extensions
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0x2F800 && r <= 0x2FA1F) {
			count++
		}
	}
	return count
}

// hasCyrillicCharacters checks for Cyrillic alphabet characters
func hasCyrillicCharacters(data []byte) bool {
	for _, r := range string(data) {
		if (r >= 0x0400 && r <= 0x04FF) || // Cyrillic
			(r >= 0x0500 && r <= 0x052F) || // Cyrillic Supplement
			(r >= 0x2DE0 && r <= 0x2DFF) || // Cyrillic Extended-A
			(r >= 0xA640 && r <= 0xA69F) { // Cyrillic Extended-B
			return true
		}
	}
	return false
}

// hasExcessiveControlChars checks if data has too many control characters
func hasExcessiveControlChars(data []byte) bool {
	controlCount := 0
	for _, b := range data {
		// Count control characters (excluding common whitespace)
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			controlCount++
		}
	}
	// More than 5% control characters is excessive
	return float64(controlCount)/float64(len(data)) > 0.05
}
