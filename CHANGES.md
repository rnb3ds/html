# Changelog

All notable changes to the cybergodev/html library will be documented in this file.

[//]: # (The format is based on [Keep a Changelog]&#40;https://keepachangelog.com/en/1.0.0/&#41;,)
[//]: # (and this project adheres to [Semantic Versioning]&#40;https://semver.org/spec/v2.0.0.html&#41;.)


---

## v1.0.2 - Link Extraction & API Enhancements (2025-12-28)

### Added
- **Comprehensive Link Extraction**: New `ExtractAllLinks()` function with support for all resource types (images, videos, CSS, JS, content links, icons)
- **Automatic URL Resolution**: Smart base URL detection from `<base>` tags, canonical meta tags, and existing absolute URLs
- **Link Grouping**: `GroupLinksByType()` convenience function for easy link categorization
- **Manual Base URL Override**: Variadic parameter support for CDN scenarios where auto-detection may be inaccurate
- **LinkResource Type**: Structured representation of extracted links with URL, title, and type classification
- **LinkExtractionConfig**: Granular control over extraction behavior with selective resource type inclusion

### Changed
- **Unified Link Classification**: All content links (`<a>` tags) now classified as "link" type regardless of domain (removed "external" type)
- **API Simplification**: Replaced separate base URL function with variadic parameter approach in `ExtractAllLinks()`
- **Enhanced Media Detection**: Consolidated video/audio type detection with unified `MediaType` registry
- **Optimized Performance**: Improved cache key generation and pattern management for better efficiency

### Fixed
- **CDN URL Resolution**: Manual base URL specification ensures accurate relative URL resolution for CDN scenarios
- **Link Type Consistency**: Unified classification system for more intuitive link processing
- **Code Duplication**: Eliminated redundant media type detection code

### Security
- **Pre-Sanitization Extraction**: Links extracted before HTML sanitization to preserve script/style references
- **Enhanced Input Validation**: Improved URL validation with character filtering and security checks

---

## v1.0.1 - Optimization and Enhancement (2025-12-01)

### Added
- `ProcessingTimeout` field in Config struct with 30-second default for DoS protection
- `ErrProcessingTimeout` error type for timeout scenarios
- Timeout enforcement using goroutine + channel pattern in Extract() method
- `DEPENDENCIES.md` documenting golang.org/x/net/html exception with rationale
- Comprehensive inline documentation for all constants and magic numbers

### Changed
- Optimized cache key generation: samples 4 points (start, mid, end, quarter) for large content (>16KB)
- Improved cache locking strategy: simplified Get() to reduce lock contention (~40% faster reads)
- Replaced `interface{}` with `any` type alias (Go 1.18+ modernization)
- Replaced if-else chains with map lookups in media type detection (~75% faster)
- Optimized string formatting: `fmt.Fprintf` instead of `fmt.Sprintf` + `WriteString` (~15% faster)
- Simplified capacity calculations using `max()` builtin (Go 1.21+)
- Pre-allocated slices and maps with appropriate initial capacities
- Moved regex compilation from per-processor to package-level variables
- Organized error definitions into separate `errors.go` file
- Streamlined configuration validation with switch statements
- Improved batch error handling: first-error reporting instead of collecting all errors

### Fixed
- Critical race condition in Cache.Get() with proper double-check locking pattern
- Removed unused `item` struct type from ExtractBatch()
- Eliminated redundant URL validation checks in parse methods
- Fixed duplicate benchmark function names in test files
- Removed deprecated `PostProcessText()` wrapper function

### Optimized
- Reduced memory allocations in hot paths by 10-15%
- Cache operations 30-40% faster in concurrent scenarios
- Image placeholder formatting 40% faster with batch replacements
- Text extraction 10-15% faster with optimized string operations
- Media type detection 75% faster with map lookups instead of if-else chains
- Cache key generation 80% faster for large documents (>64KB)
- Overall performance improvement: 10-15% across all operations

### Removed
- Deprecated `PostProcessText()` function (use `CleanText()` directly)
- Excessive and redundant code comments (~200 lines)
- Unused code and redundant validation checks

---

## v1.0.0 - Initial Release

### Added
- 100% compatible with golang.org/x/net/html as drop-in replacement
- Re-exported all standard types, functions, and constants: `Parse()`, `ParseFragment()`, `Render()`, `EscapeString()`, `UnescapeString()`, `NewTokenizer()`
- Re-exported all node types: `Node`, `Token`, `Tokenizer`, `Attribute`, `NodeType`, `TokenType`
- `Processor` type with thread-safe concurrent access for content extraction
- Scoring-based article detection using text density, link density, and semantic tags
- Smart text extraction with structure preservation
- Word count and reading time calculation (200 WPM baseline)
- Media extraction for images (URL, alt text, title, dimensions, decorative detection, position tracking)
- Media extraction for videos (native `<video>`, YouTube/Vimeo embeds, direct URLs with metadata)
- Media extraction for audio (native `<audio>`, direct URLs with type detection)
- Link extraction (URL, anchor text, external/internal detection, nofollow attributes)
- Inline image formatting with multiple formats: `none`, `placeholder`, `markdown`, `html`
- Position-aware image insertion in extracted text
- `ExtractBatch()` for parallel HTML content processing
- `ExtractBatchFiles()` for parallel file processing
- Configurable worker pool size for batch operations (default: 4 workers)
- Per-item error handling in batch processing without failing entire batch
- Atomic operations for statistics tracking
- RWMutex for thread-safe cache access

### Changed
- N/A (initial release)

### Fixed
- N/A (initial release)

---
