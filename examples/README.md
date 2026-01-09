# 📚 Examples - Learning Path

This directory contains comprehensive examples demonstrating all features of the `cybergodev/html` package.

### 1️⃣ [01_quick_start.go](01_quick_start.go) - Quick Start
**Start here!** The simplest introduction to the `cybergodev/html` package.

**What you'll learn:**
- Extract content with default settings
- Use convenience functions: `Extract()`, `ExtractText()`, `ExtractFromFile()`
- Basic processor usage for multiple operations

**Run it:**
```bash
go run -tags examples examples/01_quick_start.go
```

---

### 2️⃣ [02_content_extraction.go](02_content_extraction.go) - Content Extraction
Comprehensive content extraction features for real-world scenarios.

**What you'll learn:**
- Smart article detection (auto-removes navigation, ads, sidebar, footer)
- Inline image formats: none, placeholder, markdown, html
- Minimal extraction (text only)
- Working with realistic blog post HTML

**Run it:**
```bash
go run -tags examples examples/02_content_extraction.go
```

---

### 3️⃣ [03_link_extraction.go](03_link_extraction.go) - Link Extraction
Extract and organize all types of links from HTML documents.

**What you'll learn:**
- Simple link extraction with defaults
- Group links by type (CSS, JS, images, videos, etc.)
- Selective extraction (filter by resource type)
- Custom base URL resolution
- Handle CDN scenarios

**Run it:**
```bash
go run -tags examples examples/03_link_extraction.go
```

---

### 4️⃣ [04_media_extraction.go](04_media_extraction.go) - Media Extraction
Extract images, videos, audio, and links with complete metadata.

**What you'll learn:**
- Extract images with dimensions and alt text
- Extract videos with poster and type information
- Extract audio resources
- Extract links with external/nofollow detection

**Run it:**
```bash
go run -tags examples examples/04_media_extraction.go
```

---

### 5️⃣ [05_advanced_usage.go](05_advanced_usage.go) - Advanced Usage
Master advanced features for production use.

**What you'll learn:**
- Custom processor configuration (timeouts, cache, workers)
- Custom extraction configuration
- Caching performance optimization
- Batch processing with worker pools
- Concurrent usage (thread-safe operations)
- Statistics monitoring

**Run it:**
```bash
go run -tags examples examples/05_advanced_usage.go
```

---

### 6️⃣ [06_compatibility.go](06_compatibility.go) - Compatibility
100% compatibility with golang.org/x/net/html standard library.

**What you'll learn:**
- All standard APIs: Parse, Render, Escape, Unescape, Tokenizer
- Node creation and manipulation
- Drop-in replacement for golang.org/x/net/html
- Enhanced content extraction (bonus features)

**Run it:**
```bash
go run -tags examples examples/06_compatibility.go
```

---

**Happy coding! 🎉**

