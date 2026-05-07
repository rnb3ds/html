# Changelog

All notable changes to the cybergodev/html library will be documented in this file.

---

## v1.4.1 - Security Hardening, Performance & Race Fix (2026-05-07)

### Security
- `AllowedBaseDir` config field restricts file operations to paths under a specified directory
- `truncateAuditURL` helper caps data URLs at 256 chars in audit logs, preventing disk exhaustion
- `FileError.MarshalJSON` uses `SafePath()` to prevent raw filesystem path disclosure in JSON responses

### Performance
- Single-pass HTML parse with in-place DOM sanitization (~41% faster on large documents)
- Direct string scanning replaces regex-based link placeholder matching in `formatInlineLinks`
- Inlined `compressAndTrimRight` into `CleanText`, eliminating nested builder pool overhead
- Pooled `NodeSlicePool` for traversal stack in `CleanContentNode`

### Fixed
- `ChannelAuditSink` Write/Close race condition — replaced done+Once with RWMutex+isClosed
- Cache hit returns deep copy via `cloneResult()` to prevent data races on concurrent access

### Added
- `SanitizeDOM` function for in-place DOM sanitization
- DOM-path tests for iframe/embed/object video extraction
- CSS injection prevention tests (`expression`/`behavior`/`-moz-binding`/`javascript`/`vbscript`)
- In-place DOM sanitization test suite (script removal, event handler stripping, URI/style sanitization)

### Changed
- Removed unused `sanitizeContent` method and `linkPlaceholderRegex` variable
- Clarified `configMu` mutex comment to reflect its narrow scope
- Documented `MarshalJSON` asymmetry — JSON format is for external consumption, not round-tripping

---

## v1.4.0 - Production Readiness & Performance (2026-04-29)

### Breaking
- `ExtractBatch`/`ExtractBatchFiles` return `*BatchResult` instead of `([]*Result, error)` — use `.Results`, `.Success`, `.Failed` fields
- 8 fine-grained interfaces consolidated into composite `Extractor` + `StatsProvider` with unified method sets
- `SetMaxSampleSize` returns void — method chaining removed per project conventions
- `compat.go` removed; all wrappers (`filepathClean`, `readFile`, `now`, `since`) replaced by stdlib calls

### Added
- `*WithContext` variants for text, Markdown, JSON, batch, and link extraction (16 new methods/functions)
- Package-level batch functions: `ExtractBatch`, `ExtractBatchWithContext`, `ExtractBatchFiles`, `ExtractBatchFilesWithContext`
- `cachekey.go` with xxHash-style cache key generation extracted from extract.go
- GitHub Actions CI workflow (vet, format check, race tests, golangci-lint)
- `SharedDefaultScorer()` singleton — eliminates ~54 allocations per `New()` call
- CSS sanitization (`sanitizeStyleValue`) stripping `expression()`, `behavior:`, `-moz-binding:`, `javascript:`, `vbscript:`
- `escapeMarkdownText()` preventing Markdown injection via unescaped `]`, `[`, `\` in alt/link text
- `sanitizeRawValue()` HTML-escaping audit `RawValue` fields to prevent XSS in downstream renderers
- `ChannelAuditSink.DroppedCount()` monitoring dropped audit entries when buffer is full
- `uniformErrorBatch` helper preserving `len(Errors) == len(Results)` invariant
- `Cache.RestartCleanup()` for correct pool processor reuse after `Close()`
- `maxBatchSize` constant (10,000) with early rejection preventing OOM on extreme input
- Comprehensive panic protection test suite (32 tests) and boundary tests for uncovered functions
- `generateCacheKey`/`hashMixInline` helpers with 128-bit xxHash-style output
- `withProcessor[T]` and `withProcessorBatch` generic helpers eliminating delegation boilerplate

### Fixed
- `ExtractWithContext` silently discarded caller context when `ProcessingTimeout > 0`
- `BatchResult.Errors` length mismatch on config resolution failure (broke index correspondence)
- Temporary processors shared parent's stats pointer, inflating `totalProcessed`/`cacheMisses`
- Unbounded goroutine spawning in batch (up to 10,000) — now bounded by `WorkerPoolSize`
- Race condition: temp processors sharing `audit` with parent caused `ChannelAuditSink` panic
- `FileError.Unwrap()` always returned `ErrInvalidFilePath` for non-NotFound errors
- Empty `mediaType` index-out-of-range panic in `extractMediaLink`
- Nil audit pointer dereference with fallback pooled processors
- Cache cleanup goroutine not restarted on processor reuse
- `CacheCleanup < 0` not validated in `Config.Validate()`
- Panic in `withTimeout` goroutine escaping caller's `recover()` — goroutine panics not caught by parent recover
- `WriterAuditSink.Write` silently discarded write errors
- Stale closed atomic flag on pooled processor reuse
- Data race on `cleanupCancel` field between `StopCleanup` and finalizer goroutine
- `ChannelAuditSink` panic on write to closed channel (added `done` check)
- `activeTimeoutGoroutines` TOCTOU race — replaced load-then-add with atomic CAS loop
- `PutBuilder` corrupting slice header during secure clear
- `ExtractAllLinksWithContext` timeout path not incrementing `errorCount`
- Format-specific results polluting parent cache through temp processors
- `isDangerousScheme` bypassed by fullwidth Unicode characters — added `normalizeFullwidthToASCII`
- Duplicate manual `stringsToLower` implementation in errors.go
- Audit `RawValue` XSS: HTML-escape `<`, `>`, `"`, `'`, `&` for downstream log renderers
- Inconsistent HTML size limit in media tag extraction (`maxHTMLForRegex*10` → `maxHTMLForRegex`)
- Unsafe `sync.Map` type assertion in `normalizeCharset()` — added ok check
- WaitGroup counter corruption on pooled processor reuse — added `audit.Wait()` before `ClearAuditLog`
- Package godoc example used non-existent `WithCache(1000, time.Hour)` API
- README false "drop-in replacement for golang.org/x/net/html" claims corrected
- `docs/COMPATIBILITY.md` rewritten — corrected all method signatures, type names, defaults, and code examples
- `docs/SECURITY.md` fixed algorithm description (SHA-256 → xxHash), constants, unsafe locations, and code samples
- Examples: build tag collisions resolved, redundant `\n` fixed, compilation errors corrected, API signatures updated

### Changed
- `Extract` delegates to `extractCoreWithContext(context.Background())`, eliminating ~80% code duplication
- Consolidated 4 `recover*` functions into single generic `recoverPanic[T any]`; old names preserved as thin wrappers
- Replaced ~70 individual on* handler entries with single `strings.HasPrefix("on")` prefix check
- Merged `DetectAndConvertToUTF8String` + `Safe` variant into shared `detectAndConvertToUTF8StringCore` (~80 lines removed)
- `resolveConfig` returns `(Config, bool, error)` — pool decision at call site, eliminating `reflect.DeepEqual` in hot path
- Merged `extractImages`/`extractImagesWithPosition` and `extractLinks`/`extractLinksWithPosition` into unified methods
- Pre-compute normalized `imageFormat`/`linkFormat` strings in `Processor`, eliminating per-call `ToLower`+`TrimSpace`
- Pool `EncodingDetector` instances via `sync.Pool` for reduced allocation in non-ASCII path
- Cache `strings.ToLower(n.Data)` result for tag name lookup in `sanitizeNodeWithAudit`
- Removed dead `processContent`, `useDefaultScorer` branch, `extractCore` passthrough, `collectResults` helper
- `processorStats` extracted to separate struct for accurate shared stats between main and temp processors
- Examples: unique per-file build tags, `interface{}` → `any`, fixed timing demos, proper error checking
- Test coverage improved from 78.9% to 79.7%; consolidated 31 test functions into 6 table-driven

### Performance
- `BenchmarkNew`: -18.8% latency, -4.6% memory, -69.2% allocs (78→24)
- `BenchmarkProcessorCreationWithClose`: -19.3% latency, -16.8% memory, -42.5% allocs (127→73)
- `countWords`: zero-allocation inline whitespace scanning replacing `strings.Fields`
- `processContent`: fast-path whitespace check avoids `TrimSpace` allocation on ASCII content
- `GetTextContent`: single-pass combining normalize + entity decode + newline + trim
- `CleanText`: early-exit detection for clean text skips `ReplaceHTMLEntities`/`unwantedCharReplacer`
- `compressAndTrimRight`: combines whitespace compression + trailing trim in one pass
- Tag removal: O(1) map lookup replacing linear slice scan in `sanitizeNode`
- Table rendering: pre-computed 32-entry padding table eliminates `strings.Repeat` per cell
- `formatInlineImages`: skips `strings.NewReplacer` allocation when no replacements needed
- Removed `reflect` import from processor_pool.go — no reflection in hot path

### Removed
- `compat.go` — all wrappers inlined to stdlib equivalents
- `BytesToStringSafe`/`StringToBytesSafe` dead code from `internal/unsafe.go`
- `isDefaultConfig`, `getProcessorWithConfig`, `putProcessorWithConfig` (reflection overhead)
- `collectResults` helper superseded by `*BatchResult` return
- `extractImages`/`extractLinks` (without position) superseded by position-tracking variants

---

## v1.3.2 - Security Hardening & Performance Enhancement (2026-03-23)

### Breaking Changes
- Removed 11 deprecated package-level `*With()` functions — use `New(Config)` + processor methods instead
- Removed `NewWithConfigs()` public function — use `New(Config)` internally

### Added
- `BytesToStringSafe()` / `StringToBytesSafe()` for safe memory conversions in untrusted contexts
- `SetPoolSecureClear()` for optional secure buffer clearing (prevents data leakage)
- `ExtractWith()` family: 10 new package-level functions with optional `Config` parameter
- `maxWalkDepth` constant (50,000 nodes) to prevent memory exhaustion attacks

### Changed
- Cache key generation uses 5-point sampling with xxHash-style hashing (~15% faster)
- `isPureASCII()` has defensive bounds checking for small slices
- `WalkNodes()` limited to 50,000 nodes maximum depth
- `replaceNumericEntity()` validates hex/decimal characters and limits to 10 chars
- `IsValidURL()` trims whitespace before protocol validation

### Fixed
- Race condition in `ExtractToMarkdown()` — now uses config copy instead of shared state
- Panic in processor pool replaced with graceful fallback
- Protocol-relative URL validation bypass via leading whitespace
- Context cancellation checked at all processing stages
- Cache cleared when returning processor to pool

### Performance
- Large document extraction: ~15-19% faster
- Hash function uses xxHash-inspired algorithm with 4 parallel accumulators
- Builder pool capacity increased from 256 to 1024 bytes
- Pattern scoring eliminates heap allocation with fixed-size stack array

### Security
- Defense-in-depth for fast path vulnerabilities (S-06 to S-16)
- Enhanced numeric entity validation prevents DoS via long strings
- Improved cache key collision resistance (5-point sampling)
- Security documentation added to Cache struct and config comparison

---

## v1.3.1 - Performance & API Enhancement (2026-03-04)

### Added
- Optional `Config` parameter for all package-level functions (backward compatible)

### Changed
- Examples restructured with new `04_performance.go` focused on batch processing and caching
- Inlined timeout handling and optimized scorer nil checks for cleaner code

### Fixed
- Missing documentation for `TextOnlyConfig()` function

### Performance
- `Extract`: ~51% faster (430 → 212 ns/op)
- `ExtractWithCache`: ~43% faster (189 → 107 ns/op)
- `ExtractLargeDocument`: ~62% faster (55000 → 21000 ns/op)
- `CleanText`: ~15% less memory, ~22% fewer allocations

---

## v1.3.0 - Performance & API Enhancement (2026-03-03)

### Added
- `LinkFormat` configuration for inline link formatting (markdown, html, none)
- `CacheCleanup` configuration with automatic background cleanup for TTL entries
- `StartCleanup()` and `StopCleanup()` methods for proactive cache management
- `SetPoolLogger()` function for pool corruption debug logging
- Optional variadic configuration parameters for `New()` constructor
- Smart config merging for `ExtractConfig` and `LinkExtractionConfig`
- Automatic cache goroutine cleanup via `runtime.SetFinalizer`
- `Len()` method to get current cache entry count
- `FromFile` variant methods to `Extractor` and `LinkExtractor` interfaces

### Changed
- `WalkNodes` converted from recursive to iterative (prevents stack overflow on deep DOM)
- `isPureASCII` optimized with 64-bit batch processing (16% CPU hotspot reduced)
- Cache key hash length increased from 8 to 16 bytes (better collision resistance)
- `Cache.Get()` uses read-write lock separation for better concurrent performance
- `ExtractToMarkdown()` now uses `DefaultConfig()` for API consistency
- `DefaultScorer` uses lazy initialization with `sync.Once`
- Examples restructured from 9 to 8 focused files

### Fixed
- Potential cache goroutine leak when Cache is garbage collected
- TOCTOU race condition in `Cache.Get()` method
- Potential nil pointer dereference in `NewDefaultScorerWithConfig()`
- Goroutine leak in `withTimeout()` with maximum limit protection
- Test error handling issues (unchecked errors, nil pointer access)

### Performance
- `Extract`: ~26% faster
- `ExtractWithCache`: ~34% faster
- `ExtractLargeDocument`: ~22% faster
- `CleanText`: ~68% faster (replaced regex with manual scanning)
- `ConcurrentExtract`: ~29% faster
- Memory allocations reduced by 50-65% in key benchmarks

### Security
- Library confirmed fully thread-safe (100+ race detection iterations)
- All shared state properly synchronized with appropriate primitives

### Breaking Changes
- Destructive update!

---

## v1.2.0 - Comprehensive Quality & Documentation Enhancement (2026-02-07)

### Breaking Changes
- **Removed**: Deprecated `NewWithDefaults()` method (use `New()` or `New(html.DefaultConfig())`)
- **Removed**: Non-existent `ExtractWithDefaults()` method from documentation
- **API Signatures**: Batch/link extraction functions now use variadic parameters (`configs ...ExtractConfig`)

### Added - Features
- **Namespace Tag Support** (P1): Comprehensive inline namespace tag detection for SEC/XBRL documents (`ix:nonnumeric`, `xbrl:value`, etc.) with proper whitespace preservation
- **HTML5 Block Elements**: Added support for `<article>`, `<section>`, `<nav>`, `<aside>`, `<header>`, `<footer>`, `<figure>`, `<figcaption>`, `<details>`, `<summary>`
- **Custom Tag Structure Awareness**: Intelligent extraction for custom/namespace tags based on actual content structure (not predefined lists)
- **Markdown Table Indentation**: Proper indentation preservation for nested tables in list items
- **New Examples** (10 total, reorganized):
  - `03_links_and_urls.go` - Comprehensive link/URL handling
  - `04_media_extraction.go` - Focused media files extraction
  - `05_config_performance.go` - Configuration & performance tuning guide
  - `06_http_integration.go` - HTTP integration patterns for web scraping
  - `09_error_handling.go` - Robust error handling patterns

### Improved - Performance (15-20% overall)
- **Encoding Detection**: Pre-compiled regex patterns, removed sync.Once lazy initialization
- **String Operations**: Reduced redundant ToLower conversions throughout codebase
- **Memory Allocation**: Optimized hot paths with pre-calculated capacities
- **Cache Performance**: Lazy eviction for expired entries, reduced system calls
- **Batch Processing**: 2-4x faster for multiple documents with worker pool pattern

### Improved - Security
- **Path Traversal Protection**: Enhanced validation in `ExtractFromFile()` with stricter checks
- **CSS Injection Protection**: Added CSS value validation in style attributes
- **Protocol Validation**: Enhanced URI protocol validation for dangerous schemes
- **ReDoS Protection**: Added protection against regex denial-of-service attacks
- **Null Byte Prevention**: Null byte injection prevention in URLs/paths

### Improved - Code Quality
- **Centralized Constants**: Created `internal/constants.go` for all internal package constants
- **URL Utilities**: Created `internal/url.go` with 6 centralized functions (`IsExternalURL`, `ExtractDomain`, `ResolveURL`, etc.)
- **Dead Code Removal**: Removed redundant functions, unused variables, duplicate code
- **Integer Overflow Fix**: Fixed potential overflow in `replaceNumericEntity`
- **Package Consistency**: Fixed `default_config_test.go` package inconsistency (black-box testing)

### Improved - Test Suite
- **New Tests** (+1,078 lines):
  - `concurrency_test.go` (430 lines): Thread safety, memory pressure, cache eviction
  - `security_test.go` (460 lines): XSS prevention, path traversal, DoS prevention
  - `testutil/testutil.go` (280 lines): Reusable test fixtures and helpers
- **Removed**: Debug-only tests without assertions (`extraction_debug_test.go`, `extraction_sec_test.go`)

### Changed - API
- **Processor Statistics**: Added `ResetStatistics()` method
- **Variadic Parameters**: All batch/link functions now accept variadic config parameters
- **Function Signatures**:
  ```go
  // Before
  processor.ExtractBatch(contents [][]byte, config ExtractConfig)

  // After (config is now variadic)
  processor.ExtractBatch(contents [][]byte)
  // or
  processor.ExtractBatch(contents [][]byte, configs ...ExtractConfig)
  ```

### Fixed - Bugs
- **Inline Element Spacing**: Fixed depth tracking for proper whitespace between inline elements
- **Namespace Tag Detection**: Fixed inline namespace tags incorrectly treated as block elements
- **Trailing Space Preservation**: Enhanced preservation with namespace tag awareness
- **Image Metadata**: Fixed `img.Src` to correct `img.URL` field reference in tests
- **Filter Function**: Removed unused return value from `filterExpandedColumns`

### Migration Guide

#### Removed `NewWithDefaults()`
```go
// Before (deprecated)
processor, _ := html.NewWithDefaults()

// After
processor, _ := html.New()
// or
processor, _ := html.New(html.DefaultConfig())
```

#### Removed `ExtractWithDefaults()`
```go
// Before (method doesn't exist)
result, _ := processor.ExtractWithDefaults(htmlBytes)

// After
result, _ := processor.Extract(htmlBytes)
// or
result, _ := processor.Extract(htmlBytes, html.DefaultExtractConfig())
```

#### Variadic Parameters
```go
// Before
processor.ExtractBatch(docs, config)

// After (config is now variadic, but backward compatible)
processor.ExtractBatch(docs, config)      // works
processor.ExtractBatch(docs)              // uses defaults
processor.ExtractBatch(docs, config1)     // single config
```

### Performance Benchmarks
- Text Extraction: ~500ns per HTML document
- Link Extraction: ~2μs per HTML document
- Full Extraction: ~5μs per HTML document
- Cache Hit: ~100ns
- **Encoding Detection**: 15-20% faster

---

## v1.1.1 - Critical Bug Fixes & Security Enhancements (2026-02-02)

### Fixed
- **Critical: Pattern Matching Word Boundary Detection**
  - Fixed false positive pattern matching causing incorrect element removal
  - Elements like `<section class="section-heading">` were incorrectly treated as ads (contained "ad")
  - Implemented proper word boundary detection with separators: `-`, `_`, space, tab
  - Text extraction from affected pages increased by 1,273x (87 → 111,010 characters)

- **Test Output Formatting**
  - Fixed `fmt.Printf` misuse that caused format errors with `%` characters
  - Prevented `%!f(MISSING)`, `%!a(MISSING)` errors in test output

- **Cache Double-Check Locking Race Condition**
  - Fixed potential race condition in cache Get method
  - Properly re-checks entry after acquiring write lock

- **HTML Entity Parsing Logic**
  - Simplified numeric entity validation (removed redundant parsing)
  - Eliminated unnecessary validation loops and goto statements

- **URI Security Validation**
  - Reordered checks to block dangerous protocols first (javascript:, vbscript:, file:)
  - Fixed potential bypass through leading/trailing whitespace
  - Corrected data URL character validation (was rejecting valid UTF-8)

### Changed
- **Code Quality**
  - Simplified re-exported types and constants (25 → 14 lines)
  - Removed unused re-exports: `Tokenizer`, `ParseOption`, `ParseWithOptions`, etc.
  - Cleaned up redundant comments throughout codebase
  - Maintained 100% backward compatibility

### Security
- Enhanced protocol validation order for safer URL handling
- Fixed data URL validation to properly handle base64-encoded content
- Corrected cache concurrency issues for thread-safe operation

### Migration Notes
- **Zero Breaking Changes** - All existing API calls work without modification
- **Tests**: All existing tests pass successfully

---

## v1.1.0 - Table Extraction Enhancement & Documentation Update (2026-02-01)

### Added
- **Table Extraction Features**:
  - Colspan expansion for Markdown tables with proper structure preservation
  - HTML format support with original colspan structure maintained
  - Visual alignment with automatic column width calculation
  - Column width preservation from both `style` and `width` attributes
  - Structure row detection (rows with width definitions only)
  - Multi-line text normalization in table cells
  - Support for all CSS text alignment values (left, center, right, justify)
  - Alignment detection from all rows (not just header)
- **Stdlib Compatibility**:
  - 100% API coverage with golang.org/x/net/html
  - Re-exported all ParseOption types and constants

### Changed
- **Text Extraction**:
  - Paragraph spacing optimization (double newlines for Markdown)
  - Inline element text extraction with multi-line handling
  - Improved HTML entity decoding
- **Examples**:
  - Restructured from 12 to 8 progressive examples
  - Added quick start guide
  - Added real-world use cases
  - Improved error handling demonstrations
- **Code Quality**:
  - Eliminated over-engineering and redundant comments
  - Removed magic numbers, added named constants
  - Enhanced input validation and security
  - Improved variable naming throughout

### Fixed
- **Critical Bugs**:
  - TableFormat cache key generation bug
  - HTML format colspan preservation
  - Structure row detection issues
  - Mixed alignment column handling
  - Data URI size limit (100KB max)
- **Documentation**:
  - Processor Methods API reference
  - LinkExtractionConfig completeness
  - Result structure JSON field names

### Performance
- Optimized large document handling (3MB+)
- Reduced allocations in text extraction
- Improved cache key generation
- Enhanced memory pooling

### Test Coverage
- Added comprehensive table extraction tests
- Enhanced URL validation tests
- Improved edge case handling

### Security
- Enhanced data URL validation
- Early input size validation
- Improved DoS prevention
- Safe HTML entity handling

### Migration Notes
- **Zero Breaking Changes** - All existing API calls work without modification
- **New Features** - Table extraction enhancements are opt-in via `TableFormat` config
- **Tests** - All previously failing tests now pass

---

## v1.0.6 - Critical Fixes & Quality Improvements (2026-01-19)

### Fixed
- **Cache Eviction Logic**
  - Fixed cache overflow issue - cache now properly respects maxEntries limit in all scenarios
  - Previously could grow indefinitely when updating existing keys
- **Test Compilation**
  - Fixed undefined function call in `internal/extraction_test.go`
- **URL Handling**
  - Fixed `normalizeBaseURL` to correctly skip non-HTTP protocol URLs (data:, javascript:, mailto:)
- **Documentation Accuracy**
  - Corrected `ExtractFromFile` API signature (was missing `configs` parameter)
  - Added missing fields to type definitions (ImageInfo.Position, LinkInfo.Title)
  - Added complete type definitions for VideoInfo, AudioInfo, LinkResource
  - Updates in both README.md and README_zh-CN.md

### Added
- New `extractTagAttributes()` helper function for parsing tag attributes from raw HTML content
- Supports quoted and unquoted attribute values with case-insensitive matching

### Changed
- **Enhanced Video Extraction** - Three-stage process:
  1. Parse iframe/embed/object from raw HTML (before sanitization)
  2. Walk DOM tree for `<video>` tags and survivors
  3. Use regex for direct video URLs in HTML
- **Optimized Cache Key Generation** - Reduced allocations with direct byte slice construction

### Security
- HTML sanitization maintained - removes iframe, embed, object tags for security
- Videos extracted before sanitization to preserve media information

### Performance
- Optimized cache key generation (fewer allocations)
- Minimal performance impact from raw HTML parsing (only when needed)

### Migration Notes
- **Zero Breaking Changes** - All existing API calls work without modification
- **Tests**: All previously failing tests now pass (TestIframeExtraction, TestEmbedExtraction)

---

## v1.0.5 - Code Quality & Maintainability Enhancement (2026-01-14)

### Fixed
- **Critical Performance Issues**:
  - Removed unnecessary mutex locking on read-only maps (significant concurrency improvement)
  - Fixed InlineImageFormat and PreserveImages parameter coupling (now independent)
  - Simplified cache eviction logic for predictable behavior
- **Security**:
  - Enhanced data URL validation (safe ASCII only, blocks injection characters)
  - Early input size validation (moved to function entry for DoS prevention)

### Changed
- **Code Quality**:
  - Eliminated backward compatibility wrappers and duplicate functions
  - Consolidated CleanText functions (single unified API)
  - Removed duplicate regex definitions (single source of truth)
  - Removed over-engineering and redundant comments (~43 lines removed)
- **Modernization**:
  - Eliminated init() functions (declaration-time initialization)
  - Simplified cache key generation (start/end segments only)
  - Removed unnecessary memory copies in JSON generation
- **API Consistency**:
  - All extraction methods now accept optional config parameters
  - Extract(), ExtractFromFile(), ExtractBatch(), ExtractBatchFiles(), ExtractAllLinks()
  - Unified LinkExtractionConfig across package-level and Processor methods

### Performance
- **Concurrency**: Removed read locks on immutable maps (major speedup)
- **Memory**: Reduced allocations with simplified text cleaning
- **Cache**: Simplified key generation (maintains 99% distribution)
- **API**: Cleaner, more consistent function signatures

### Removed
- Redundant wrapper functions (ensureNewline, ensureSpacing, extractTable wrapper)
- Duplicate function definitions and regex patterns
- Over-commented code (kept only valuable documentation)
- Deprecated writeJSONString function

### Migration Notes
- **Zero Breaking Changes**: All existing API calls work without modification
- **Optional Configs**: New optional parameters use variadic syntax (backward compatible)
- **Behavior Change**: InlineImageFormat and PreserveImages now work independently

---

## v1.0.4 - Thread-Safety & Performance Optimization (2026-01-12)

### Fixed
- **CRITICAL: Thread-Safety Issues**:
  - Fixed concurrent map access causing runtime panics in production environments
  - Added `sync.RWMutex` protection for all global scoring and media pattern maps
  - Fixed cache race conditions with proper locking patterns in `Get()` and `evictOne()`
  - Eliminated all data races detected by race detector
- **Performance Optimizations**:
  - Zero-allocation text extraction using `trackedBuilder` pattern (eliminated millions of string allocations)
  - Optimized JSON generation with `sync.Pool` and efficient string building (~50-70% faster)
  - Implemented memory pooling for reduced GC pressure
  - Performance improvements: Extract() ~83% faster, ExtractToJSON() ~15% faster

### Added
- **Package-Level Convenience API** (17 new functions):
  - Format conversion: `ExtractToMarkdown()`, `ExtractToJSON()`
  - Quick extraction: `ExtractText()`, `ExtractTitle()`, `ExtractImages()`, `ExtractVideos()`, `ExtractAudios()`, `ExtractLinks()`
  - Content analysis: `GetWordCount()`, `GetReadingTime()`, `Summarize()`
  - Text processing: `ExtractAndClean()`, `ExtractWithTitle()`
  - Configuration presets: `ConfigForRSS()`, `ConfigForSearchIndex()`, `ConfigForSummary()`, `ConfigForMarkdown()`
- **Comprehensive Test Coverage**:
  - Increased test coverage from 64.5% to 77.8%
  - Added 200+ new test cases
  - All package-level functions fully tested
  - Concurrent stress tests: 295,852 operations with 0 errors

### Changed
- **Regex Operations**: Removed unnecessary mutex overhead
- **Cache Implementation**: Improved lock contention handling and eviction strategy
- **Code Quality**:
  - Improved variable naming throughout (descriptive names instead of single letters)
  - Enhanced code documentation with performance notes
  - Simplified complex code patterns for better maintainability

### Security
- **XSS Protection**: Fixed XSS vulnerability in HTML output with proper escaping
- **Input Validation**: Reduced MaxInputSize from 1GB to 50MB for better DoS protection
- **Thread-Safety**: All shared state properly synchronized for concurrent use

### Performance
- **Text Extraction**: 83% faster (2,800 → 460 ns/op)
- **JSON Generation**: 15% faster with 60% fewer allocations
- **Memory Usage**: 90% reduction in allocations (4,500 → 448 B/op)
- **Cache Operations**: 5-10% faster under high concurrency load
- **Scalability**: Production-ready for high-throughput concurrent processing

### Migration Notes
- **Zero Breaking Changes**: 100% API compatible
- **All Changes Internal**: Existing code continues to work without modification

---

## v1.0.3 - Performance & Quality Optimization (2026-01-09)

### Changed
- **Performance Improvements**:
  - Pattern matching: O(n) → O(1) lookup complexity using hash maps
  - Base URL detection: 75% reduction in DOM traversals
  - Cache eviction: O(2n) → O(n) single-pass algorithm
  - Media type detection: O(n) → O(1) with map-based lookup
- **Code Quality**:
  - Consolidated constants from 7 to 3 (57% reduction)
  - Reduced redundant comments (~30% reduction)
  - Enhanced function documentation
- **Test Suite**: Consolidated test files (38% reduction in root, 14% in internal)
- **Examples**: Reduced from 12 to 6 examples (50% reduction)

### Fixed
- **Data URI Support**: Fixed link extraction for data URIs with special characters
- **Scoring Logic**: Corrected weakNegativeScore from -300 to -100
- **Hidden Element Detection**: Enhanced display:none and visibility:hidden detection
- **Documentation**: Fixed all example file references

### Security
- Enhanced URL validation and DoS prevention (50MB max input)

---

## v1.0.2 - Link Extraction & API Enhancements (2025-12-28)

### Added
- **Comprehensive Link Extraction**: `ExtractAllLinks()` with automatic URL resolution
- **Link Grouping**: `GroupLinksByType()` convenience function
- **LinkResource Struct**: URL, title, and type classification
- **LinkExtractionConfig**: Granular control over extraction behavior

### Changed
- **Unified Link Classification**: All `<a>` tags now "link" type
- **Enhanced Media Detection**: Consolidated video/audio type detection

### Security
- **Pre-Sanitization Extraction**: Links extracted before sanitization
- **Enhanced Input Validation**: Improved URL validation with security checks

---

## v1.0.1 - Optimization and Enhancement (2025-12-01)

### Added
- `ProcessingTimeout` field with 30-second default for DoS protection
- `ErrProcessingTimeout` error type
- `DEPENDENCIES.md` documentation

### Changed
- Optimized cache key generation (4-point sampling for large content)
- Improved cache locking (~40% faster reads)
- Replaced `interface{}` with `any` (Go 1.18+)
- Optimized media type detection with map lookups (~75% faster)
- Replaced regex compilation with package-level variables

### Fixed
- Critical race condition in Cache.Get()
- Removed deprecated functions

### Optimized
- Reduced memory allocations 10-15%
- Cache operations 30-40% faster
- Overall performance improvement 10-15%

---

## v1.0.0 - Initial Release

### Added
- 100% compatible with golang.org/x/net/html as drop-in replacement
- Re-exported all standard types and functions
- `Processor` type with thread-safe concurrent access
- Scoring-based article detection
- Smart text extraction with structure preservation
- Media extraction (images, videos, audio)
- Link extraction with metadata
- Inline image formatting (none, placeholder, markdown, html)
- `ExtractBatch()` for parallel processing
- Configurable worker pool (default: 4 workers)
- Atomic operations and RWMutex for thread-safety

---
