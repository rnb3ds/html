# Security Policy

## Overview

The `github.com/cybergodev/html` library is designed with security as a core principle. This document outlines the security measures, threat model, and best practices for using this library safely in production environments.

## Security Architecture

### Defense-in-Depth Strategy

The library implements multiple layers of security controls:

1. **Input Validation** - All inputs validated at API boundaries
2. **Resource Limits** - Configurable limits prevent resource exhaustion
3. **Safe Parsing** - Uses battle-tested `golang.org/x/net/html` parser
4. **Content Sanitization** - Removes dangerous content by default
5. **Thread Safety** - Concurrent access protected with proper synchronization
6. **Error Handling** - No panics, all errors returned explicitly

## Threat Model

### Protected Against

#### 1. Denial of Service (DoS)

**Input Size Attacks**
- **Threat**: Malicious actors submit extremely large HTML documents to exhaust memory
- **Mitigation**: `MaxInputSize` limit (default: 50MB)
- **Validation**: Input size checked before processing
- **Error**: Returns `ErrInputTooLarge` when exceeded

```go
config := html.Config{
    MaxInputSize: 10 * 1024 * 1024, // 10MB limit
}
processor, _ := html.New(config)
```

**Deeply Nested HTML (Billion Laughs Attack)**
- **Threat**: Exponentially nested elements cause stack overflow or excessive processing
- **Mitigation**: `MaxDepth` limit (default: 100 levels)
- **Validation**: DOM depth validated during parsing
- **Error**: Returns `ErrMaxDepthExceeded` when exceeded

```go
config := html.Config{
    MaxDepth: 50, // Limit nesting to 50 levels
}
```

**Processing Timeout**
- **Threat**: Complex HTML causes indefinite processing
- **Mitigation**: `ProcessingTimeout` (default: 30 seconds)
- **Validation**: Operations respect context deadlines
- **Error**: Returns `ErrProcessingTimeout` when exceeded

**Cache Exhaustion**
- **Threat**: Unique inputs fill cache memory
- **Mitigation**: `MaxCacheEntries` limit with LRU eviction
- **Validation**: Cache size enforced with automatic eviction
- **Cleanup**: Lazy TTL-based expiration

#### 2. Code Injection

**Script Injection (XSS)**
- **Threat**: Malicious `<script>` tags execute in downstream applications
- **Mitigation**: `EnableSanitization` removes dangerous tags and attributes
- **Default**: Enabled by default
- **Scope**: Content removed before parsing
- **Removed Tags**: `<script>`, `<style>`, `<noscript>`, `<iframe>`, `<embed>`, `<object>`, `<form>`, `<input>`, `<button>`

```go
config := html.Config{
    EnableSanitization: true, // Default: true
}
```

**Event Handler Injection**
- **Threat**: Inline event handlers execute JavaScript in downstream applications
- **Mitigation**: All event handler attributes removed during sanitization
- **Removed Attributes**: `onclick`, `onerror`, `onload`, `onmouseover`, `onmouseout`, `onfocus`, `onblur`, `onchange`, `onsubmit`, `onreset`, `ondblclick`
- **Scope**: Event handlers not included in extracted content

**Dangerous URI Schemes**
- **Threat**: `javascript:`, `vbscript:`, `data:` URLs execute code
- **Mitigation**: URI validation removes dangerous schemes
- **Blocked Schemes**: `javascript:`, `vbscript:`, `file:`
- **Validated Schemes**: `data:` URLs validated for size (100KB max) and safe content

#### 3. Resource Exhaustion

**Memory Exhaustion**
- **Threat**: Large documents or cache consume all available memory
- **Mitigation**:
  - Input size limits (50MB default)
  - Cache size limits with LRU eviction
  - Efficient string builders with pre-allocated capacity
  - No unbounded allocations

**Regex DoS (ReDoS)**
- **Threat**: Malicious input causes catastrophic backtracking in regex
- **Mitigation**:
  - All regex patterns pre-compiled at initialization
  - Regex only applied to size-limited content (`maxHTMLForRegex = 1MB`)
  - Match limits enforced (`maxRegexMatches = 100`)
  - Simple, non-backtracking patterns used

```go
// Regex only applied when safe
if len(htmlContent) <= maxHTMLForRegex {
    matches := p.videoRegex.FindAllString(htmlContent, maxRegexMatches)
}
```

**URL Length Attacks**
- **Threat**: Extremely long URLs in attributes cause memory issues
- **Mitigation**: URL length limited to 2000 characters (`maxURLLength`)
- **Validation**: URLs exceeding limit are silently ignored

**Data URL Attacks**
- **Threat**: Large data URLs exhaust memory
- **Mitigation**: Data URL length limited to 100KB (`maxDataURILength`)
- **Validation**: Data URLs exceeding limit are rejected

#### 4. Cache Poisoning

**Hash Collision Attacks**
- **Threat**: Crafted inputs produce same cache key, causing incorrect results
- **Mitigation**: SHA-256 hash for cache keys (cryptographically secure)
- **Collision Probability**: Negligible (2^-256)

```go
// Cache key generation with SHA-256
hasher := sha256.New()
hasher.Write([]byte{flags})  // Config flags
hasher.Write([]byte(format))  // Format options
hasher.Write([]byte(content)) // Content (sampled for large content)
cacheKey := hex.EncodeToString(hasher.Sum(nil))
```

**Cache Timing Attacks**
- **Threat**: Timing differences reveal cached vs. non-cached content
- **Mitigation**: Not applicable - library is not cryptographic
- **Note**: Cache hits are observable through statistics API (by design)

### Not Protected Against

#### 1. Malicious HTML Content

**Phishing Content**
- **Scope**: Library extracts content as-is
- **Responsibility**: Application must validate extracted URLs and text
- **Recommendation**: Implement URL allowlists/blocklists in application layer

**Misleading Links**
- **Scope**: Library preserves link text and URLs without validation
- **Responsibility**: Application must verify link destinations
- **Detection**: Use `LinkInfo.IsExternal` to identify external links

#### 2. Privacy Concerns

**Tracking Pixels**
- **Scope**: Image URLs extracted without filtering
- **Responsibility**: Application must filter tracking domains
- **Detection**: Check `ImageInfo.Width` and `ImageInfo.Height` for 1x1 images

**Third-Party Embeds**
- **Scope**: Video/audio embed URLs extracted without validation
- **Responsibility**: Application must validate embed sources
- **Detection**: Use `VideoInfo.Type == "embed"` to identify embeds

#### 3. Content Validation

**Inappropriate Content**
- **Scope**: Library does not filter offensive or inappropriate text
- **Responsibility**: Application must implement content moderation

**Copyright Violations**
- **Scope**: Library extracts all content without rights verification
- **Responsibility**: Application must respect copyright and licensing

## Security Best Practices

### 1. Configuration Hardening

**Production Configuration**
```go
config := html.Config{
    MaxInputSize:       5 * 1024 * 1024,  // 5MB - adjust based on use case
    ProcessingTimeout:  10 * time.Second,  // Fail fast
    MaxCacheEntries:    500,               // Limit memory usage
    CacheTTL:           30 * time.Minute,  // Expire stale entries
    WorkerPoolSize:     4,                 // Limit concurrency
    EnableSanitization: true,              // Always enable
    MaxDepth:           50,                // Prevent deep nesting
}
```

**Untrusted Input Configuration**
```go
config := html.Config{
    MaxInputSize:       1 * 1024 * 1024,  // 1MB - strict limit
    ProcessingTimeout:  5 * time.Second,   // Very short timeout
    MaxCacheEntries:    100,               // Small cache
    CacheTTL:           5 * time.Minute,   // Short TTL
    EnableSanitization: true,              // Critical
    MaxDepth:           30,                // Conservative depth
}
```

### 2. Input Validation

**Always Validate Before Processing**
```go
func processUserHTML(userInput string) (*html.Result, error) {
    // 1. Validate input is not empty
    if strings.TrimSpace(userInput) == "" {
        return nil, errors.New("empty input")
    }

    // 2. Check size before passing to processor
    if len(userInput) > 10*1024*1024 {
        return nil, errors.New("input too large")
    }

    // 3. Optional: Validate HTML structure
    if !strings.Contains(userInput, "<") {
        return nil, errors.New("not HTML content")
    }

    // 4. Process with configured limits
    return processor.Extract(userInput, config)
}
```

### 3. Error Handling

**Never Ignore Errors**
```go
result, err := processor.Extract(htmlContent, config)
if err != nil {
    // Log error with context
    log.Printf("HTML extraction failed: %v", err)

    // Handle specific errors
    if errors.Is(err, html.ErrInputTooLarge) {
        return nil, fmt.Errorf("document too large: %w", err)
    }
    if errors.Is(err, html.ErrMaxDepthExceeded) {
        return nil, fmt.Errorf("document too complex: %w", err)
    }
    if errors.Is(err, html.ErrProcessingTimeout) {
        return nil, fmt.Errorf("processing timeout: %w", err)
    }

    return nil, err
}
```

### 4. Resource Management

**Always Close Processor**
```go
processor := html.NewWithDefaults()
defer processor.Close() // Critical: releases cache and resources

result, err := processor.Extract(htmlContent, config)
// ... use result
```

**Monitor Resource Usage**
```go
stats := processor.GetStatistics()
log.Printf("Cache hit rate: %.2f%%",
    float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100)

// Clear cache if memory pressure detected
if stats.CacheHits < stats.CacheMisses {
    processor.ClearCache()
}
```

### 5. Output Sanitization

**Sanitize Extracted Content**
```go
result, err := processor.Extract(htmlContent, config)
if err != nil {
    return err
}

// Sanitize URLs before use
for _, link := range result.Links {
    if !isAllowedDomain(link.URL) {
        log.Printf("Blocked suspicious URL: %s", link.URL)
        continue
    }
    // Use link.URL
}

// Filter tracking pixels
for _, img := range result.Images {
    if img.Width == "1" && img.Height == "1" {
        log.Printf("Skipping tracking pixel: %s", img.URL)
        continue
    }
    // Use img.URL
}
```

### 6. Concurrent Usage

**Thread-Safe by Design**
```go
// Single processor can be safely shared across goroutines
processor := html.NewWithDefaults()
defer processor.Close()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(content string) {
        defer wg.Done()
        result, err := processor.Extract(content, config)
        // ... handle result
    }(htmlContents[i])
}
wg.Wait()
```

**Batch Processing**
```go
// Use built-in batch processing for efficiency
results, err := processor.ExtractBatch(htmlContents, config)
if err != nil {
    // Partial failures are reported in error
    log.Printf("Batch processing: %v", err)
}

// Check individual results
for i, result := range results {
    if result == nil {
        log.Printf("Item %d failed", i)
        continue
    }
    // Process result
}
```

## Dependency Security

### Minimal Attack Surface

**Single External Dependency**
- `golang.org/x/net/html` - Official Go supplementary network libraries
- Maintained by Go team
- Battle-tested HTML5 parser
- No transitive dependencies

**Dependency Verification**
```bash
# Verify dependencies
go mod verify

# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Update to latest secure version
go get -u golang.org/x/net/html
go mod tidy
```

### Supply Chain Security

**Module Verification**
```bash
# Enable Go module checksum database
export GOSUMDB=sum.golang.org

# Verify module authenticity
go mod download
go mod verify
```

## Vulnerability Reporting

### Reporting a Vulnerability

If you discover a security vulnerability, please follow responsible disclosure:

1. **Do Not** open a public GitHub issue
2. **Email** security concerns to: [security contact - to be added]
3. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### Response Timeline

- **24 hours**: Initial acknowledgment
- **7 days**: Preliminary assessment
- **30 days**: Fix development and testing
- **Coordinated disclosure**: After fix is released

### Security Updates

Security fixes are released as:
- **Patch versions** (1.0.x) for minor issues
- **Minor versions** (1.x.0) for moderate issues
- **Immediate patches** for critical vulnerabilities

## Security Testing

### Automated Testing

The library includes comprehensive security-focused tests with **85%+ code coverage**:

```bash
# Run all tests including security tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run concurrency stress tests
go test -v -run TestStress

# Run with coverage
go test -cover ./...
```

### Security Test Coverage

- Input validation tests (size limits, depth limits, timeout)
- XSS prevention tests (sanitization effectiveness)
- Resource limit tests (cache, memory, allocations)
- Concurrent access tests (race conditions, thread safety)
- URL validation tests (dangerous schemes, length limits)
- Data URL validation tests (size limits, content validation)

### Fuzzing

**Recommended Fuzzing Targets**
```go
// Fuzz input validation
func FuzzExtract(f *testing.F) {
    processor := html.NewWithDefaults()
    defer processor.Close()

    f.Fuzz(func(t *testing.T, data []byte) {
        processor.Extract(string(data), html.DefaultExtractConfig())
        // Should never panic
    })
}
```

### Security Audit Checklist

- [x] Input validation on all public APIs
- [x] Resource limits enforced
- [x] No panics in production code
- [x] All errors properly handled
- [x] Thread-safe concurrent access
- [x] No unsafe package usage
- [x] No arbitrary code execution
- [x] Regex patterns are safe
- [x] Cache keys are cryptographically secure (SHA-256)
- [x] Dependencies are up to date

## Compliance

### OWASP Top 10 (2021)

| Risk | Status | Mitigation |
|------|--------|------------|
| A01: Broken Access Control | N/A | Library does not handle authentication |
| A02: Cryptographic Failures | ✅ Protected | SHA-256 for cache keys |
| A03: Injection | ✅ Protected | Content sanitization, no code execution |
| A04: Insecure Design | ✅ Protected | Defense-in-depth architecture |
| A05: Security Misconfiguration | ✅ Protected | Secure defaults, validation |
| A06: Vulnerable Components | ✅ Protected | Minimal dependencies, regular updates |
| A07: Authentication Failures | N/A | Library does not handle authentication |
| A08: Software/Data Integrity | ✅ Protected | Module verification, checksums |
| A09: Logging Failures | ⚠️ Partial | Application must implement logging |
| A10: Server-Side Request Forgery | N/A | Library does not make network requests |

### CWE Coverage

- **CWE-20**: Input Validation - ✅ Comprehensive validation
- **CWE-79**: XSS - ✅ Content sanitization with tag/attribute removal
- **CWE-89**: SQL Injection - N/A (no database access)
- **CWE-119**: Buffer Overflow - ✅ Go memory safety
- **CWE-190**: Integer Overflow - ✅ Validated limits
- **CWE-400**: Resource Exhaustion - ✅ Resource limits (DoS protection)
- **CWE-770**: Allocation without Limits - ✅ Size limits enforced
- **CWE-835**: Infinite Loop - ✅ Depth limits, timeouts

## Security Changelog

### v1.0.7 (2026-02-01)
- Enhanced table extraction with colspan and alignment support
- Improved input validation for data URLs (100KB limit)
- Enhanced sanitization with additional dangerous attributes
- Test coverage increased to 85%+

### v1.0.6 (2026-01-19)
- Fixed cache eviction logic to respect maxEntries
- Enhanced URL validation for non-HTTP protocols
- Improved documentation accuracy

### v1.0.5 (2025-01-14)
- Enhanced data URL validation (safe ASCII only, blocks injection)
- Early input size validation (moved to function entry)
- Improved concurrent access patterns

### v1.0.4 (2026-01-12)
- **CRITICAL**: Fixed thread-safety issues (concurrent map access)
- Eliminated all data races detected by race detector
- Added comprehensive concurrent stress tests

### Initial Release (v1.0.0)
- Input validation on all APIs
- Resource limits with configurable defaults
- Content sanitization enabled by default
- Thread-safe concurrent access
- SHA-256 cache keys
- Comprehensive security testing

## Additional Resources

- [Go Security Best Practices](https://golang.org/doc/security)
- [OWASP HTML Sanitization](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Go Race Detector](https://golang.org/doc/articles/race_detector.html)

## License

This security policy is part of the `github.com/cybergodev/html` project and is licensed under the same terms as the project.

---
