package internal

import (
	"testing"
)

// Benchmarks for entity replacement performance

var (
	// Text with only common entities (fast path)
	benchCommonEntities = "This is a test &nbsp; with &amp; common &lt; entities &gt; like &quot; quotes &apos; and &copy; copyright &reg; registered &mdash; dash &ndash; end."

	// Text with mixed common and rare entities
	benchMixedEntities = "Test &nbsp; text &amp; with &euro; euro &pound; pound &yen; yen &sect; section &para; paragraph &plusmn; plusminus &times; multiply &divide; divide &frac12; half &deg; degree &micro; micro &middot; dot &bull; bullet &dagger; dagger &permil; permille."

	// Text with numeric entities
	benchNumericEntities = "Text with &#65; &#x41; &#160; &#xa0; &#8212; &#x2014; &#169; &#xa9; numeric &#8364; &#x20ac; entities."

	// Text with no entities (should be very fast)
	benchNoEntities = "This is just plain text without any HTML entities at all in it. Should be very fast to process."

	// Long text with common entities
	benchLongCommon = repeatString("Test &nbsp; text &amp; with &lt; entities &gt; and ", 100)
)

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func BenchmarkReplaceHTMLEntities_CommonEntities(b *testing.B) {
	b.SetBytes(int64(len(benchCommonEntities)))
	for i := 0; i < b.N; i++ {
		ReplaceHTMLEntities(benchCommonEntities)
	}
}

func BenchmarkReplaceHTMLEntities_MixedEntities(b *testing.B) {
	b.SetBytes(int64(len(benchMixedEntities)))
	for i := 0; i < b.N; i++ {
		ReplaceHTMLEntities(benchMixedEntities)
	}
}

func BenchmarkReplaceHTMLEntities_NumericEntities(b *testing.B) {
	b.SetBytes(int64(len(benchNumericEntities)))
	for i := 0; i < b.N; i++ {
		ReplaceHTMLEntities(benchNumericEntities)
	}
}

func BenchmarkReplaceHTMLEntities_NoEntities(b *testing.B) {
	b.SetBytes(int64(len(benchNoEntities)))
	for i := 0; i < b.N; i++ {
		ReplaceHTMLEntities(benchNoEntities)
	}
}

func BenchmarkReplaceHTMLEntities_LongText(b *testing.B) {
	b.SetBytes(int64(len(benchLongCommon)))
	for i := 0; i < b.N; i++ {
		ReplaceHTMLEntities(benchLongCommon)
	}
}

// Benchmark fastReplaceCommonEntities directly
func BenchmarkFastReplaceCommonEntities(b *testing.B) {
	b.SetBytes(int64(len(benchCommonEntities)))
	for i := 0; i < b.N; i++ {
		fastReplaceCommonEntities(benchCommonEntities)
	}
}

func BenchmarkFastReplaceCommonEntities_LongText(b *testing.B) {
	b.SetBytes(int64(len(benchLongCommon)))
	for i := 0; i < b.N; i++ {
		fastReplaceCommonEntities(benchLongCommon)
	}
}
