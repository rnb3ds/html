# Changelog

All notable changes to the cybergodev/html library will be documented in this file.

[//]: # (The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/))
[//]: # (and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html))

---

## v1.0.5 - Code Quality & Maintainability Enhancement (2025-01-14)

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
