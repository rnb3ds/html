package internal

import (
	htmlstd "html"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	// cleanTextGrowthFactor is the factor used to estimate the size of cleaned text.
	// Cleaning typically reduces text size by removing extra whitespace, so we use
	// half the original text length as an initial capacity estimate.
	cleanTextGrowthFactor = 2
	// builderInitialSize is the initial capacity for strings.Builder in text extraction functions.
	builderInitialSize = 256
)

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
func normalizeNonBreakingSpaces(s string) string {
	return strings.ReplaceAll(s, "\u00a0", " ")
}

var defaultWhitespaceRegex = regexp.MustCompile(`\s+`)

var unwantedCharReplacer = strings.NewReplacer(
	"☒", "[X]",
	"☐", "[ ]",
	"☑", "[X]",
)

func CleanText(text string, whitespaceRegex *regexp.Regexp) string {
	if text == "" {
		return ""
	}
	if whitespaceRegex == nil {
		whitespaceRegex = defaultWhitespaceRegex
	}
	textLen := len(text)
	var result strings.Builder
	result.Grow(textLen / cleanTextGrowthFactor)
	start := 0
	previousWasEmpty := false

	for i := 0; i <= textLen; i++ {
		if i == textLen || text[i] == '\n' {
			line := whitespaceRegex.ReplaceAllString(text[start:i], " ")
			isEmpty := true
			if line != "" {
				if line = strings.TrimSpace(line); line != "" {
					isEmpty = false
					if result.Len() > 0 {
						if previousWasEmpty {
							result.WriteByte('\n')
						}
						result.WriteByte('\n')
					}
					result.WriteString(line)
				}
			}
			previousWasEmpty = isEmpty
			start = i + 1
		}
	}
	return ReplaceHTMLEntities(unwantedCharReplacer.Replace(result.String()))
}

func WalkNodes(node *html.Node, fn func(*html.Node) bool) {
	if node == nil || fn == nil || !fn(node) {
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		WalkNodes(child, fn)
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
	var sb strings.Builder
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
func fastReplaceCommonEntities(text string) string {
	// Quick check: if none of the common entity patterns exist, return early
	if !strings.Contains(text, "&amp;") &&
		!strings.Contains(text, "&nbsp;") &&
		!strings.Contains(text, "&lt;") &&
		!strings.Contains(text, "&gt;") &&
		!strings.Contains(text, "&quot;") &&
		!strings.Contains(text, "&apos;") &&
		!strings.Contains(text, "&copy;") &&
		!strings.Contains(text, "&reg;") &&
		!strings.Contains(text, "&mdash;") &&
		!strings.Contains(text, "&ndash;") {
		return text
	}

	var sb strings.Builder
	sb.Grow(len(text))

	i := 0
	for i < len(text) {
		if text[i] != '&' {
			sb.WriteByte(text[i])
			i++
			continue
		}

		// Check if we have enough characters for an entity
		if i+4 >= len(text) {
			sb.WriteByte(text[i])
			i++
			continue
		}

		// Fast switch for common entities (ordered by frequency)
		switch {
		case text[i:i+5] == "&amp;":
			sb.WriteByte('&')
			i += 5
		case text[i:i+6] == "&nbsp;":
			sb.WriteByte(' ')
			i += 6
		case text[i:i+4] == "&lt;":
			sb.WriteByte('<')
			i += 4
		case text[i:i+4] == "&gt;":
			sb.WriteByte('>')
			i += 4
		case text[i:i+6] == "&quot;":
			sb.WriteByte('"')
			i += 6
		case text[i:i+6] == "&apos;":
			sb.WriteByte('\'')
			i += 6
		case text[i:i+6] == "&copy;":
			sb.WriteString("©")
			i += 6
		case text[i:i+5] == "&reg;":
			sb.WriteString("®")
			i += 5
		case text[i:i+7] == "&mdash;":
			sb.WriteString("—")
			i += 7
		case text[i:i+7] == "&ndash;":
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
	result := strings.Builder{}
	result.Grow(len(text))

	i := 0
	for i < len(text) {
		if text[i] != '&' {
			result.WriteByte(text[i])
			i++
			continue
		}

		// Find the end of the entity (semicolon or end of string)
		end := i + 1
		if end >= len(text) {
			result.WriteByte(text[i])
			break
		}

		// Check if this is a numeric entity
		if text[end] == '#' {
			replaced, consumed := replaceNumericEntity(text, i)
			result.WriteString(replaced)
			i += consumed
			continue
		}

		// For non-numeric entities, find the semicolon
		semi := strings.IndexByte(text[i:], ';')
		if semi == -1 {
			// No semicolon found, write the '&' and continue
			result.WriteByte(text[i])
			i++
			continue
		}
		semi += i

		// Extract entity name
		entityName := text[i+1 : semi]

		// Validate entity name (alphanumeric only)
		if !isValidEntityName(entityName) {
			result.WriteByte(text[i])
			i++
			continue
		}

		// Try to decode using standard library for unknown entities
		// This handles HTML5 named entities not in our replacer
		decoded := decodeEntityFallback("&" + entityName + ";")
		result.WriteString(decoded)
		i = semi + 1
	}

	return result.String()
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

	// Parse the number
	num, err := strconv.ParseInt(entity, base, 32)
	if err != nil || num < 0 || num > 0x10FFFF {
		// Invalid numeric entity, return as-is
		return text[start : semi+1], semi - start + 1
	}

	// Check for surrogate pairs and invalid Unicode code points
	if num >= 0xD800 && num <= 0xDFFF {
		// Surrogate pair, not valid as a standalone character
		return "\uFFFD", semi - start + 1
	}

	// Special handling: convert non-breaking space (0xa0) to regular space (0x20)
	// This ensures consistent behavior with named entity &nbsp; which maps to regular space
	if num == 0xA0 {
		return " ", semi - start + 1
	}

	// Valid Unicode character
	return string(rune(num)), semi - start + 1
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

func IsExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "//")
}

// IsValidURL checks if a URL is valid and safe for processing.
// This is a centralized URL validation function exported for use across the package.
const (
	maxURLLength     = 2000
	maxDataURILength = 100000
)

func IsValidURL(url string) bool {
	urlLen := len(url)
	if urlLen == 0 || urlLen > maxURLLength {
		return false
	}

	// Special handling for data URLs - stricter validation with size limit
	if strings.HasPrefix(url, "data:") {
		if urlLen > maxDataURILength {
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

	// Accept absolute and protocol-relative URLs
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return true
	}
	if strings.HasPrefix(url, "//") {
		return true
	}

	// Accept relative URLs and paths (starting with / or .)
	if url[0] == '/' || url[0] == '.' {
		return true
	}

	// Accept alphanumeric paths (legitimate filenames like img1.jpg, video.mp4)
	// but reject paths starting with special characters that might be used in injection attacks
	if urlLen > 0 {
		firstChar := url[0]
		if (firstChar >= 'a' && firstChar <= 'z') ||
			(firstChar >= 'A' && firstChar <= 'Z') ||
			(firstChar >= '0' && firstChar <= '9') {
			return true
		}
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
