# cybergodev/html - Release Notes

All notable changes to this project will be documented in this file.

======================================================================


## [v1.0.0] - Initial version

### Added

**100% Compatible with golang.org/x/net/html**
- Drop-in replacement with all types, functions, and constants re-exported
- `Parse()`, `ParseFragment()`, `Render()`, `EscapeString()`, `UnescapeString()`, `NewTokenizer()`
- All node types: `Node`, `Token`, `Tokenizer`, `Attribute`, `NodeType`, `TokenType`

**Intelligent Content Extraction**
- `Processor` type with thread-safe concurrent access
- Scoring-based article detection (text density, link density, semantic tags)
- Smart text extraction with structure preservation
- Word count and reading time calculation (200 WPM)

**Media Extraction**
- Images: URL, alt text, title, dimensions, decorative detection, position tracking
- Videos: Native `<video>`, YouTube/Vimeo embeds, direct URLs with metadata
- Audio: Native `<audio>`, direct URLs with type detection
- Links: URL, anchor text, external/internal detection, nofollow attributes

**Inline Image Formatting**
- Multiple formats: `none`, `placeholder`, `markdown`, `html`
- Position-aware image insertion in extracted text
- Configurable via `InlineImageFormat` in `ExtractConfig`

**Batch Processing**
- `ExtractBatch()` for parallel HTML content processing
- `ExtractBatchFiles()` for parallel file processing
- Configurable worker pool size (default: 4 workers)
- Per-item error handling without failing entire batch

**Thread Safety**
- All `Processor` methods safe for concurrent use
- No external synchronization required
- Atomic operations for statistics
- RWMutex for cache access

### Dependencies
- Single dependency: `golang.org/x/net v0.47.0`
- Go 1.24+ required

---
