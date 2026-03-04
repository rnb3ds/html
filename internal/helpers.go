package internal

import (
	htmlstd "html"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Pre-compiled regex patterns for text processing.
// These are compile-time constants and will panic on initialization if invalid.
var (
	paddingLeftRegex = regexp.MustCompile(`padding-left:\s*(\d+(?:\.\d+)?)\s*pt`)
)

// Clean text and builder sizing constants are now in constants.go

// boundaryCharSets defines different character sets for word boundary detection
type boundaryCharSet int

const (
	// boundaryStandard uses standard word boundary characters (-, _, space, tab)
	boundaryStandard boundaryCharSet = iota
	// boundaryCSS uses CSS-specific boundary characters (;, :, space, tab, {, }, ")
	boundaryCSS
)

// hasWordBoundary checks if a pattern appears with proper word boundaries.
// The allowed boundary characters are determined by the boundaryCharSet parameter.
func hasWordBoundary(text, pattern string, charSet boundaryCharSet) bool {
	idx := strings.Index(text, pattern)
	if idx == -1 {
		return false
	}

	// Check character before the match
	if idx > 0 {
		before := text[idx-1]
		if !isBoundaryChar(before, charSet) {
			return false
		}
	}

	// Check character after the match
	endIdx := idx + len(pattern)
	if endIdx < len(text) {
		after := text[endIdx]
		if !isBoundaryChar(after, charSet) {
			return false
		}
	}

	return true
}

// isBoundaryChar checks if a character is a valid boundary character
func isBoundaryChar(c byte, charSet boundaryCharSet) bool {
	switch charSet {
	case boundaryCSS:
		return c == ';' || c == ':' || c == ' ' || c == '\t' ||
			c == '{' || c == '}' || c == '"'
	case boundaryStandard:
		return c == '-' || c == '_' || c == ' ' || c == '\t'
	default:
		return false
	}
}

// normalizeNonBreakingSpaces replaces all non-breaking spaces (\u00a0) with regular spaces.
// This ensures consistent text processing across different HTML sources.
// Optimized with early exit to avoid allocation when no NBSP is present.
func normalizeNonBreakingSpaces(s string) string {
	// Fast path: early exit if no non-breaking spaces present
	// This avoids the allocation overhead of strings.ReplaceAll for common case
	if !strings.Contains(s, "\u00a0") {
		return s
	}
	return strings.ReplaceAll(s, "\u00a0", " ")
}

// normalizeLineBreaks replaces newlines with spaces and removes carriage returns
// in a single pass. This is more efficient than two separate ReplaceAll calls.
// Optimized with combined detection and processing in a single allocation.
func normalizeLineBreaks(s string) string {
	// Fast path: single scan to detect if any processing is needed
	// and find the first position that needs modification
	n := len(s)
	firstMod := -1
	for i := 0; i < n; i++ {
		if s[i] == '\n' || s[i] == '\r' {
			firstMod = i
			break
		}
	}

	// No line breaks or carriage returns found
	if firstMod == -1 {
		return s
	}

	// Single pass replacement starting from first modification point
	sb := GetBuilder()
	defer PutBuilder(sb)
	sb.Grow(n)

	// Copy unchanged prefix
	if firstMod > 0 {
		sb.WriteString(s[:firstMod])
	}

	// Process from first modification point
	for i := firstMod; i < n; i++ {
		c := s[i]
		if c == '\n' {
			sb.WriteByte(' ')
		} else if c == '\r' {
			// Skip carriage returns
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

var unwantedCharReplacer = strings.NewReplacer(
	"☒", "[X]",
	"☐", "[ ]",
	"☑", "[X]",
)

// compressWhitespace compresses multiple whitespace characters to single space
// without using regex. Returns the compressed string.
// Optimized with single-pass detection and processing using pooled builder.
func compressWhitespace(s string) string {
	n := len(s)
	if n == 0 {
		return ""
	}

	// Single scan to find first position needing compression or tab conversion
	firstMod := -1
	needsSpace := false // Track if previous char was space/tab
	for i := 0; i < n; i++ {
		c := s[i]
		if c == ' ' || c == '\t' {
			if needsSpace {
				// Found consecutive whitespace - needs compression
				firstMod = i - 1
				break
			}
			needsSpace = true
		} else {
			needsSpace = false
		}
		// Also check for tabs that need conversion
		if c == '\t' && firstMod == -1 {
			firstMod = i
		}
	}

	// No modification needed
	if firstMod == -1 {
		return s
	}

	// Use pooled builder for result
	sb := GetBuilder()
	defer PutBuilder(sb)
	sb.Grow(n)

	// Copy unchanged prefix
	if firstMod > 0 {
		sb.WriteString(s[:firstMod])
	}

	// Process from first modification point
	inSpace := false
	for i := firstMod; i < n; i++ {
		c := s[i]
		if c == ' ' || c == '\t' {
			if !inSpace {
				sb.WriteByte(' ')
				inSpace = true
			}
		} else {
			sb.WriteByte(c)
			inSpace = false
		}
	}

	return sb.String()
}

func CleanText(text string, whitespaceRegex *regexp.Regexp) string {
	if text == "" {
		return ""
	}

	// Fast path: check if processing is needed
	// For clean text without special characters, skip expensive processing
	hasNewlines := strings.Contains(text, "\n")
	hasMultipleSpaces := strings.Contains(text, "  ") || strings.Contains(text, "\t")
	hasNBSP := strings.Contains(text, "\u00a0")

	if !hasNewlines && !hasMultipleSpaces && !hasNBSP {
		// Clean text - just normalize entities
		return ReplaceHTMLEntities(text)
	}

	textLen := len(text)

	// Use pooled builder for better memory efficiency
	sb := GetBuilder()
	defer PutBuilder(sb)

	sb.Grow(textLen / cleanTextGrowthFactor)
	start := 0
	previousWasEmpty := false

	for i := 0; i <= textLen; i++ {
		if i == textLen || text[i] == '\n' {
			rawLine := text[start:i]
			isEmpty := true

			// Process the line while preserving leading indentation
			if rawLine != "" {
				// Compress whitespace AFTER leading indentation
				// Find the first non-space character
				firstNonSpace := 0
				for firstNonSpace < len(rawLine) && rawLine[firstNonSpace] == ' ' {
					firstNonSpace++
				}

				if firstNonSpace < len(rawLine) {
					// Has leading indentation
					indent := rawLine[:firstNonSpace]
					content := rawLine[firstNonSpace:]

					// Use optimized whitespace compression instead of regex
					content = compressWhitespace(content)
					content = strings.TrimRight(content, " ")

					if content != "" {
						line := indent + content
						isEmpty = false
						if sb.Len() > 0 {
							if previousWasEmpty {
								sb.WriteByte('\n')
							}
							sb.WriteByte('\n')
						}
						sb.WriteString(line)
					}
				} else {
					// No leading indentation, compress all whitespace
					line := compressWhitespace(rawLine)
					line = strings.TrimRight(line, " ")
					if line != "" {
						isEmpty = false
						if sb.Len() > 0 {
							if previousWasEmpty {
								sb.WriteByte('\n')
							}
							sb.WriteByte('\n')
						}
						sb.WriteString(line)
					}
				}
			}

			previousWasEmpty = isEmpty
			start = i + 1
		}
	}
	return ReplaceHTMLEntities(unwantedCharReplacer.Replace(sb.String()))
}

// WalkNodes traverses the HTML node tree iteratively using an explicit stack
// to avoid potential stack overflow on deeply nested documents.
// The fn callback is called for each node. If fn returns false, traversal
// stops for that branch (node's children are not visited).
func WalkNodes(node *html.Node, fn func(*html.Node) bool) {
	if node == nil || fn == nil {
		return
	}

	// Use explicit stack to avoid recursion on deep DOM trees
	stack := make([]*html.Node, 0, 64)
	stack = append(stack, node)

	// Pre-allocate children buffer for reuse across iterations.
	// This avoids allocating a new slice on every loop iteration.
	childrenBuf := make([]*html.Node, 0, 16)

	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !fn(n) {
			continue
		}

		// Reset buffer for reuse
		childrenBuf = childrenBuf[:0]

		// Collect children
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			childrenBuf = append(childrenBuf, child)
		}

		// Add children in reverse order for correct traversal order
		// (first child processed first when popped from stack)
		for i := len(childrenBuf) - 1; i >= 0; i-- {
			stack = append(stack, childrenBuf[i])
		}
	}
}

func FindElementByTag(doc *html.Node, tagName string) *html.Node {
	var result *html.Node
	WalkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == tagName {
			result = n
			return false
		}
		return true
	})
	return result
}

func GetTextContent(node *html.Node) string {
	// Use pooled builder for better memory efficiency
	sb := GetBuilder()
	defer PutBuilder(sb)

	sb.Grow(builderInitialSize)
	prevEndedWithSpace := false

	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			textData := normalizeNonBreakingSpaces(n.Data)
			textData = ReplaceHTMLEntities(textData)
			textData = strings.ReplaceAll(textData, "\n", " ")

			endedWithSpace := len(textData) > 0 && (textData[len(textData)-1] == ' ' || textData[len(textData)-1] == '\t')
			startedWithSpace := len(textData) > 0 && (textData[0] == ' ' || textData[0] == '\t')

			text := strings.TrimSpace(textData)
			if text != "" {
				needsSpace := prevEndedWithSpace
				if !needsSpace && sb.Len() > 0 {
					needsSpace = startedWithSpace
				}

				if sb.Len() > 0 && needsSpace {
					sb.WriteByte(' ')
				}
				sb.WriteString(text)
			}
			prevEndedWithSpace = endedWithSpace
		}
		return true
	})
	return sb.String()
}

func GetTextLength(node *html.Node) int {
	length := 0
	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			textData := normalizeNonBreakingSpaces(n.Data)
			length += len(strings.TrimSpace(textData))
		}
		return true
	})
	return length
}

func GetLinkDensity(node *html.Node) float64 {
	if node == nil {
		return 0.0
	}

	textLength := 0
	linkTextLength := 0

	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			textData := normalizeNonBreakingSpaces(n.Data)
			length := len(strings.TrimSpace(textData))
			textLength += length
			for parent := n.Parent; parent != nil; parent = parent.Parent {
				if parent.Type == html.ElementNode && parent.Data == "a" {
					linkTextLength += length
					break
				}
			}
		}
		return true
	})

	if textLength == 0 {
		return 0.0
	}
	return float64(linkTextLength) / float64(textLength)
}

var entityReplacer = strings.NewReplacer(
	// Note: Common entities (&amp;, &nbsp;, &lt;, &gt;, &quot;, &apos;, &copy;, &reg;, &mdash;, &ndash;)
	// are handled in fastReplaceCommonEntities() for better performance.

	// Remaining typographic entities
	"&hellip;", "…",
	"&trade;", "™",
	// Currency symbols
	"&euro;", "€",
	"&pound;", "£",
	"&cent;", "¢",
	"&yen;", "¥",
	"&curren;", "¤",
	// Mathematical symbols
	"&sect;", "§",
	"&para;", "¶",
	"&plusmn;", "±",
	"&times;", "×",
	"&divide;", "÷",
	"&frac12;", "½",
	"&frac14;", "¼",
	"&frac34;", "¾",
	"&deg;", "°",
	"&prime;", "'",
	"&Prime;", "\"",
	"&sup1;", "¹",
	"&sup2;", "²",
	"&sup3;", "³",
	// Additional common entities
	"&middot;", "·",
	"&bull;", "•",
	"&rsquo;", "'",
	"&lsquo;", "'",
	"&rdquo;", "\"",
	"&ldquo;", "\"",
	"&sbquo;", "‚",
	"&bdquo;", "„",
	"&dagger;", "†",
	"&Dagger;", "‡",
	"&permil;", "‰",
	"&micro;", "µ",
)

// ReplaceHTMLEntities replaces HTML entities with their corresponding characters.
// It handles both named entities (like &amp;, &nbsp;) and numeric entities (like &#65;, &#x41;).
// For unknown entities, it falls back to the standard library's html.UnescapeString.
// Optimized with a fast path for the most common entities.
func ReplaceHTMLEntities(text string) string {
	if !strings.ContainsRune(text, '&') {
		return text
	}

	// Fast path: handle the 10 most common entities directly
	// This avoids the overhead of strings.NewReplacer for the majority case
	result := fastReplaceCommonEntities(text)
	if result != text {
		// If we replaced entities, still need to handle numeric ones
		return replaceHTMLEntitiesFull(result)
	}

	// Slow path: use entityReplacer for less common entities, then handle numeric
	text = entityReplacer.Replace(text)
	return replaceHTMLEntitiesFull(text)
}

// fastReplaceCommonEntities handles the 10 most common HTML entities with direct scanning.
// This is significantly faster than strings.NewReplacer for these common cases.
// Returns the input string unchanged if no common entities were found.
// Optimized with single-pass detection to avoid multiple strings.Contains() calls.
func fastReplaceCommonEntities(text string) string {
	textLen := len(text)

	// Single scan to find first ampersand - avoids multiple Contains() calls
	firstAmpersand := -1
	for i := 0; i < textLen; i++ {
		if text[i] == '&' {
			firstAmpersand = i
			break
		}
	}

	// Fast path: no ampersands means no entities possible
	if firstAmpersand == -1 {
		return text
	}

	// Quick check if any common entity patterns exist starting from first ampersand
	// This avoids scanning the entire string for each pattern
	remaining := text[firstAmpersand:]
	hasCommonEntity := false
	for i := 0; i < len(remaining); i++ {
		if remaining[i] == '&' {
			// Check for common entity patterns at this position
			remLen := len(remaining) - i
			switch {
			case remLen >= 5 && remaining[i:i+5] == "&amp;":
				hasCommonEntity = true
			case remLen >= 6 && remaining[i:i+6] == "&nbsp;":
				hasCommonEntity = true
			case remLen >= 4 && remaining[i:i+4] == "&lt;":
				hasCommonEntity = true
			case remLen >= 4 && remaining[i:i+4] == "&gt;":
				hasCommonEntity = true
			case remLen >= 6 && remaining[i:i+6] == "&quot;":
				hasCommonEntity = true
			case remLen >= 6 && remaining[i:i+6] == "&apos;":
				hasCommonEntity = true
			case remLen >= 6 && remaining[i:i+6] == "&copy;":
				hasCommonEntity = true
			case remLen >= 5 && remaining[i:i+5] == "&reg;":
				hasCommonEntity = true
			case remLen >= 7 && remaining[i:i+7] == "&mdash;":
				hasCommonEntity = true
			case remLen >= 7 && remaining[i:i+7] == "&ndash;":
				hasCommonEntity = true
			}
			if hasCommonEntity {
				break
			}
		}
	}

	// Fast path: no common entities found
	if !hasCommonEntity {
		return text
	}

	// Use pooled builder for better memory efficiency
	sb := GetBuilder()
	defer PutBuilder(sb)

	sb.Grow(textLen)

	// Write prefix unchanged
	if firstAmpersand > 0 {
		sb.WriteString(text[:firstAmpersand])
	}

	i := firstAmpersand
	for i < textLen {
		if text[i] != '&' {
			sb.WriteByte(text[i])
			i++
			continue
		}

		// Check if we have at least 4 characters for the shortest entity (&lt;)
		remainingLen := textLen - i
		if remainingLen < 4 {
			sb.WriteByte(text[i])
			i++
			continue
		}

		// Fast switch for common entities (ordered by frequency)
		// Each case checks bounds before slicing to prevent panic
		switch {
		case remainingLen >= 5 && text[i:i+5] == "&amp;":
			sb.WriteByte('&')
			i += 5
		case remainingLen >= 6 && text[i:i+6] == "&nbsp;":
			sb.WriteByte(' ')
			i += 6
		case remainingLen >= 4 && text[i:i+4] == "&lt;":
			sb.WriteByte('<')
			i += 4
		case remainingLen >= 4 && text[i:i+4] == "&gt;":
			sb.WriteByte('>')
			i += 4
		case remainingLen >= 6 && text[i:i+6] == "&quot;":
			sb.WriteByte('"')
			i += 6
		case remainingLen >= 6 && text[i:i+6] == "&apos;":
			sb.WriteByte('\'')
			i += 6
		case remainingLen >= 6 && text[i:i+6] == "&copy;":
			sb.WriteString("©")
			i += 6
		case remainingLen >= 5 && text[i:i+5] == "&reg;":
			sb.WriteString("®")
			i += 5
		case remainingLen >= 7 && text[i:i+7] == "&mdash;":
			sb.WriteString("—")
			i += 7
		case remainingLen >= 7 && text[i:i+7] == "&ndash;":
			sb.WriteString("–")
			i += 7
		default:
			// Not a common entity, copy as-is
			sb.WriteByte(text[i])
			i++
		}
	}

	return sb.String()
}

// replaceHTMLEntitiesFull handles numeric entities and unknown named entities.
func replaceHTMLEntitiesFull(text string) string {
	// Use pooled builder for better memory efficiency
	sb := GetBuilder()
	defer PutBuilder(sb)

	sb.Grow(len(text))

	i := 0
	for i < len(text) {
		if text[i] != '&' {
			sb.WriteByte(text[i])
			i++
			continue
		}

		// Find the end of the entity (semicolon or end of string)
		end := i + 1
		if end >= len(text) {
			sb.WriteByte(text[i])
			break
		}

		// Check if this is a numeric entity
		if text[end] == '#' {
			replaced, consumed := replaceNumericEntity(text, i)
			sb.WriteString(replaced)
			i += consumed
			continue
		}

		// For non-numeric entities, find the semicolon
		semi := strings.IndexByte(text[i:], ';')
		if semi == -1 {
			// No semicolon found, write the '&' and continue
			sb.WriteByte(text[i])
			i++
			continue
		}
		semi += i

		// Extract entity name
		entityName := text[i+1 : semi]

		// Validate entity name (alphanumeric only)
		if !isValidEntityName(entityName) {
			sb.WriteByte(text[i])
			i++
			continue
		}

		// Try to decode using standard library for unknown entities
		// This handles HTML5 named entities not in our replacer
		decoded := decodeEntityFallback("&" + entityName + ";")
		sb.WriteString(decoded)
		i = semi + 1
	}

	return sb.String()
}

// replaceNumericEntity handles numeric character references like &#65; or &#x41;
func replaceNumericEntity(text string, start int) (string, int) {
	if start+2 >= len(text) || text[start+1] != '#' {
		return string(text[start]), 1
	}

	// Find the semicolon
	semi := strings.IndexByte(text[start:], ';')
	if semi == -1 {
		return string(text[start]), 1
	}
	semi += start

	entity := text[start+2 : semi]
	if len(entity) == 0 {
		return text[start : semi+1], semi - start + 1
	}

	var base int
	if entity[0] == 'x' || entity[0] == 'X' {
		base = 16
		entity = entity[1:]
	} else {
		base = 10
	}

	// Parse the number with 64-bit to prevent overflow
	num, err := strconv.ParseInt(entity, base, 64)
	if err != nil || num < 0 || num > 0x10FFFF {
		// Invalid numeric entity, return as-is
		return text[start : semi+1], semi - start + 1
	}

	// Check for surrogate pairs and invalid Unicode code points
	if num >= 0xD800 && num <= 0xDFFF {
		// Surrogate pair, not valid as a standalone character
		return "\uFFFD", semi - start + 1
	}

	// Convert to rune and validate it's a valid Unicode code point
	r := rune(num)
	if !utf8.ValidRune(r) {
		return "\uFFFD", semi - start + 1
	}

	// Special handling: convert non-breaking space (0xa0) to regular space (0x20)
	// This ensures consistent behavior with named entity &nbsp; which maps to regular space
	if num == 0xA0 {
		return " ", semi - start + 1
	}

	// Valid Unicode character
	return string(r), semi - start + 1
}

// isValidEntityName checks if an entity name contains only valid characters.
func isValidEntityName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// decodeEntityFallback attempts to decode an entity using the standard library.
// This serves as a fallback for HTML5 named entities not in our fast replacer.
func decodeEntityFallback(entity string) string {
	// The standard library handles all HTML5 named entities
	decoded := htmlstd.UnescapeString(entity)
	if decoded == entity {
		// Entity was not recognized, return as-is
		return entity
	}
	return decoded
}

// IsValidURL checks if a URL is valid and safe for processing.
// This is a centralized URL validation function with size limits for security.
func IsValidURL(url string) bool {
	urlLen := len(url)
	if urlLen == 0 || urlLen > MaxURLLength {
		return false
	}

	// Special handling for data URLs - stricter validation with size limit
	if strings.HasPrefix(url, "data:") {
		if urlLen > MaxDataURILength {
			return false
		}
		for i := 5; i < urlLen; i++ {
			b := url[i]
			if b < 32 || b > 126 || b == '<' || b == '>' || b == '"' || b == '\'' || b == '\\' {
				return false
			}
		}
		return true
	}

	// Validate non-data URLs: check for dangerous characters
	for i := 0; i < urlLen; i++ {
		b := url[i]
		if b < 32 || b == 127 || b == '<' || b == '>' || b == '"' || b == '\'' {
			return false
		}
	}

	// Check for dangerous protocol-relative URL patterns
	// Block //javascript:, //vbscript:, etc.
	if strings.HasPrefix(url, "//") {
		lowerRest := strings.ToLower(url[2:])
		if strings.HasPrefix(lowerRest, "javascript:") ||
			strings.HasPrefix(lowerRest, "vbscript:") ||
			strings.HasPrefix(lowerRest, "data:") ||
			strings.HasPrefix(lowerRest, "file:") {
			return false
		}
		return true
	}

	// Accept absolute URLs
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return true
	}

	// At this point, urlLen > 0 (verified by the urlLen == 0 check at function start)
	// Accept relative URLs and paths (starting with / or .)
	// But reject path traversal patterns
	firstChar := url[0]
	if firstChar == '/' {
		// Block path traversal attempts like /\/, /../
		if urlLen > 1 && (url[1] == '\\' || (url[1] == '.' && (urlLen == 2 || url[2] == '.' || url[2] == '/'))) {
			return false
		}
		return true
	}
	if firstChar == '.' {
		// Block directory traversal
		if strings.HasPrefix(url, "./.") || strings.HasPrefix(url, "../") {
			return false
		}
		return true
	}

	// Accept alphanumeric paths (legitimate filenames like img1.jpg, video.mp4)
	// but reject paths starting with special characters that might be used in injection attacks
	if (firstChar >= 'a' && firstChar <= 'z') ||
		(firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= '0' && firstChar <= '9') {
		return true
	}

	return false
}

func SelectBestCandidate(candidates map[*html.Node]int) *html.Node {
	var bestNode *html.Node
	bestScore := -1

	for node, score := range candidates {
		if score > bestScore {
			bestScore = score
			bestNode = node
		}
	}
	return bestNode
}

// extractPaddingLeft extracts the padding-left value from an HTML element's style attribute.
// It parses the CSS style attribute and returns the padding-left value in points (pt).
// Returns 0 if padding-left is not found or cannot be parsed.
//
// Examples:
//   - "padding-left:18pt" → 18
//   - "padding-left:63pt;" → 63
//   - "padding-left: 1.5em" → 0 (only pt is supported)
//   - "" → 0
func extractPaddingLeft(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode {
		return 0
	}

	// Get the style attribute
	var styleAttr string
	for _, attr := range node.Attr {
		if attr.Key == "style" {
			styleAttr = attr.Val
			break
		}
	}

	if styleAttr == "" {
		return 0
	}

	matches := paddingLeftRegex.FindStringSubmatch(styleAttr)
	if len(matches) < 2 {
		return 0
	}

	// Parse the numeric value
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	return int(value)
}

// getIndentLevel returns the indentation level (0-3) based on padding-left value.
func getIndentLevel(paddingLeft int) int {
	switch {
	case paddingLeft <= 18:
		return 0
	case paddingLeft <= 40:
		return 1
	case paddingLeft <= 80:
		return 2
	default:
		return 3
	}
}

// getListPrefix returns the Markdown list prefix based on padding-left value.
// This converts CSS padding-left to Markdown list nesting format.
// Example:
//   - 0-18pt   → "" (no prefix, root level)
//   - 19-40pt  → "  - " (level 1, 2 spaces indent)
//   - 41-80pt  → "    - " (level 2, 4 spaces indent)
//   - >80pt    → "      - " (level 3, 6 spaces indent)
func getListPrefix(paddingLeft int) string {
	level := getIndentLevel(paddingLeft)
	switch level {
	case 0:
		return "" // Root level, no bullet
	case 1:
		return "  - " // Level 1: 2 spaces + "- "
	case 2:
		return "    - " // Level 2: 4 spaces + "- "
	case 3:
		return "      - " // Level 3: 6 spaces + "- "
	default:
		return ""
	}
}
