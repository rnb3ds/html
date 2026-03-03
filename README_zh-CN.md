# HTML 库

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/html)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/html)

**一个用于智能 HTML 内容提取的 Go 库。** 兼容 `golang.org/x/net/html` —— 可作为直接替代品使用，并获得增强的内容提取功能。

**[📖 English Documentation](README.md)** - 英文文档

## ✨ 核心特性

### 内容提取
- **文章检测**：使用评分算法识别主要内容（文本密度、链接密度、语义标签）
- **智能文本提取**：保留结构、处理换行、计算字数和阅读时间
- **媒体提取**：提取图片、视频、音频及其元数据（URL、尺寸、alt 文本、类型检测）
- **链接分析**：外部/内部链接检测、nofollow 属性识别、锚文本提取

### 性能优势
- **内容寻址缓存**：基于 FNV-128a 的键值，支持 TTL 和 LRU 淘汰策略
- **批处理**：可配置 Worker Pool 的并行提取，支持 Context
- **线程安全**：无需外部同步即可并发使用
- **资源限制**：可配置输入大小、嵌套深度和超时保护

### 安全特性
- **HTML 净化**：移除危险标签和属性
- **审计日志**：跟踪安全事件以满足合规要求
- **输入验证**：大小限制、深度限制、路径遍历防护

### 应用场景
- **新闻聚合器**：从新闻网站提取文章内容
- **网页爬虫**：从 HTML 页面获取结构化数据
- **内容管理**：将 HTML 转换为 Markdown 或其他格式
- **搜索引擎**：索引主要内容，排除导航和广告
- **数据分析**：大规模提取和分析网页内容

---

## 📦 安装

```bash
go get github.com/cybergodev/html
```

---

## ⚡ 5 分钟快速入门

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    // 从 HTML 中提取纯文本
    htmlBytes := []byte(`
        <html>
            <nav>导航栏</nav>
            <article><h1>你好世界</h1><p>这里是内容...</p></article>
            <footer>页脚</footer>
        </html>
    `)
    text, err := html.ExtractText(htmlBytes)
    if err != nil {
        panic(err)
    }
    fmt.Println(text) // "你好世界\n这里是内容..."
}
```

**就这么简单！** 库会自动：
- 移除导航、页脚、广告
- 提取主要内容
- 清理空白字符

---

## 🚀 快速指南

### 单行函数

只想快速完成任务？使用这些包级函数：

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`<html><body><h1>标题</h1><p>内容在这里...</p></body></html>`)

    // 仅提取文本
    text, _ := html.ExtractText(htmlBytes)

    // 提取所有内容
    result, _ := html.Extract(htmlBytes)
    fmt.Println(result.Title)     // 标题
    fmt.Println(result.Text)      // 内容在这里...
    fmt.Println(result.WordCount) // 2

    // 提取所有资源链接
    links, _ := html.ExtractAllLinks(htmlBytes)

    // 格式转换
    markdown, _ := html.ExtractToMarkdown(htmlBytes)
    jsonData, _ := html.ExtractToJSON(htmlBytes)
}
```

**适用场景：** 简单脚本、一次性任务、快速原型开发

---

### 基础 Processor 使用

需要更多控制？创建一个 Processor：

```go
package main

import (
    "fmt"
    "log"
    "github.com/cybergodev/html"
)

func main() {
    // 使用默认配置创建 Processor
    processor, err := html.New()
    if err != nil {
        log.Fatal(err)
    }
    defer processor.Close()

    htmlBytes := []byte(`<html><body><h1>标题</h1><p>内容</p></body></html>`)

    // 使用默认配置提取
    result, _ := processor.Extract(htmlBytes)

    // 从文件提取
    result, _ = processor.ExtractFromFile("page.html")

    // 批量处理
    htmlContents := [][]byte{htmlBytes, htmlBytes, htmlBytes}
    results, _ := processor.ExtractBatch(htmlContents)

    fmt.Printf("已处理 %d 个文档\n", len(results))
}
```

**适用场景：** 多次提取、处理大量文件、网络爬虫

---

### 自定义配置

精细调整提取内容：

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`<html><body><h1>标题</h1><img src="img.jpg"><p>内容</p></body></html>`)

    config := html.Config{
        // 提取设置
        ExtractArticle:    true,       // 自动检测主要内容
        PreserveImages:    true,       // 提取图片元数据
        PreserveLinks:     true,       // 提取链接元数据
        PreserveVideos:    false,      // 跳过视频
        PreserveAudios:    false,      // 跳过音频
        ImageFormat:       "none",     // 选项: "none", "markdown", "html", "placeholder"
        LinkFormat:        "none",     // 选项: "none", "markdown", "html"
        TableFormat:       "markdown", // 选项: "markdown", "html"
        Encoding:          "",         // 从 meta 标签自动检测，或指定: "utf-8", "windows-1252" 等
    }

    processor, _ := html.New(config)
    defer processor.Close()

    result, _ := processor.Extract(htmlBytes)
    fmt.Printf("找到 %d 张图片\n", len(result.Images))
}
```

**适用场景：** 特定提取需求、格式转换、自定义输出

---

### 使用预设配置

库提供了便捷的预设配置：

```go
// 仅文本 - 不保留媒体
processor, _ := html.New(html.TextOnlyConfig())

// Markdown 输出 - 图片格式化为 markdown
processor, _ := html.New(html.MarkdownConfig())

// 默认 - 启用所有功能
processor, _ := html.New(html.DefaultConfig())

// 高安全 - 对不受信任的输入使用更严格的限制
processor, _ := html.New(html.HighSecurityConfig())
```

---

### 高级功能

#### 自定义 Processor 配置

```go
package main

import (
    "time"
    "github.com/cybergodev/html"
)

func main() {
    // 方法 1：使用 Config 结构体
    config := html.Config{
        MaxInputSize:       10 * 1024 * 1024, // 10MB 限制
        ProcessingTimeout:  30 * time.Second,
        MaxCacheEntries:    500,
        CacheTTL:           30 * time.Minute,
        CacheCleanup:       5 * time.Minute,  // 后台清理间隔
        WorkerPoolSize:     8,
        EnableSanitization: true,  // 移除 <script>, <style> 标签
        MaxDepth:           50,    // 防止深度嵌套攻击
    }
    processor, _ := html.New(config)
    defer processor.Close()

    // 方法 2：使用高安全预设
    processor2, _ := html.New(html.HighSecurityConfig())
    defer processor2.Close()
}
```

#### 链接提取

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`
        <html>
        <head><link rel="stylesheet" href="style.css"></head>
        <body>
            <img src="image.jpg">
            <a href="https://example.com">链接</a>
            <script src="app.js"></script>
        </body>
        </html>
    `)

    processor, _ := html.New()
    defer processor.Close()

    // 提取所有资源链接
    links, _ := processor.ExtractAllLinks(htmlBytes)

    // 按类型分组
    byType := html.GroupLinksByType(links)
    cssLinks := byType["css"]
    jsLinks := byType["js"]
    images := byType["image"]

    fmt.Printf("CSS: %d, JS: %d, 图片: %d\n", len(cssLinks), len(jsLinks), len(images))
}
```

#### 缓存与统计

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    processor, _ := html.New()
    defer processor.Close()

    htmlBytes := []byte(`<html><body><p>内容</p></body></html>`)

    // 自动启用缓存
    processor.Extract(htmlBytes)
    processor.Extract(htmlBytes) // 缓存命中！

    // 查看性能统计
    stats := processor.GetStatistics()
    fmt.Printf("缓存命中: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

    // 清除缓存（保留统计数据）
    processor.ClearCache()

    // 重置统计（保留缓存条目）
    processor.ResetStatistics()
}
```

**适用场景：** 生产应用、性能优化、特定用例

---

## 📖 常用示例

常见任务的复制粘贴解决方案：

### 提取文章文本（纯净）

```go
text, _ := html.ExtractText(htmlBytes)
// 返回不含导航/广告的纯净文本
```

### 从文件提取

```go
// 从文件提取文本
text, _ := html.ExtractTextFromFile("page.html")

// 从文件提取完整结果
result, _ := html.ExtractFromFile("page.html")

// 从文件提取链接
processor, _ := html.New()
links, _ := processor.ExtractAllLinksFromFile("page.html")

// 将文件转换为 Markdown
markdown, _ := html.ExtractToMarkdownFromFile("page.html")

// 将文件转换为 JSON
jsonData, _ := html.ExtractToJSONFromFile("page.html")
```

### 提取内容与图片

```go
result, _ := html.Extract(htmlBytes)
for _, img := range result.Images {
    fmt.Printf("图片: %s (alt: %s)\n", img.URL, img.Alt)
}
```

### 转换为 Markdown

```go
markdown, _ := html.ExtractToMarkdown(htmlBytes)
// 图片格式变为: ![alt](url)
```

### 提取所有链接

```go
processor, _ := html.New()
links, _ := processor.ExtractAllLinks(htmlBytes)
for _, link := range links {
    fmt.Printf("%s: %s\n", link.Type, link.URL)
}
```

### 获取阅读时间

```go
result, _ := html.Extract(htmlBytes)
minutes := result.ReadingTime.Minutes()
fmt.Printf("阅读时间: %.1f 分钟", minutes)
```

### 批量处理文件

```go
processor, _ := html.New()
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, _ := processor.ExtractBatchFiles(files)
```

### 带 Context 的批处理（支持取消）

```go
package main

import (
    "context"
    "time"
    "github.com/cybergodev/html"
)

func main() {
    processor, _ := html.New()
    defer processor.Close()

    files := []string{"page1.html", "page2.html", "page3.html"}

    // 创建带超时的 context
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 支持取消的处理
    result := processor.ExtractBatchFilesWithContext(ctx, files)

    fmt.Printf("成功: %d, 失败: %d, 取消: %d\n",
        result.Success, result.Failed, result.Cancelled)
}
```

### 使用预设配置

```go
// 仅文本 - 创建文本专用配置
processor, _ := html.New(html.TextOnlyConfig())
result, _ := processor.Extract(htmlBytes)

// 完整内容含 Markdown 图片
processor, _ = html.New(html.MarkdownConfig())
result, _ = processor.Extract(htmlBytes)
```

---

## 🔧 API 快速参考

### 包级函数

```go
// 提取（从字节）
html.Extract(htmlBytes []byte) (*Result, error)
html.ExtractText(htmlBytes []byte) (string, error)

// 提取（从文件）
html.ExtractFromFile(filePath string) (*Result, error)
html.ExtractTextFromFile(filePath string) (string, error)

// 格式转换（从字节）
html.ExtractToMarkdown(htmlBytes []byte) (string, error)
html.ExtractToJSON(htmlBytes []byte) ([]byte, error)

// 格式转换（从文件）
html.ExtractToMarkdownFromFile(filePath string) (string, error)
html.ExtractToJSONFromFile(filePath string) ([]byte, error)

// 链接（从字节）
html.ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error)

// 链接（从文件）
html.ExtractAllLinksFromFile(filePath string) ([]LinkResource, error)
html.GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### Processor 方法
```go
// 创建
processor, err := html.New()
processor, err := html.New(config)                    // 使用 Config 结构体
processor, err := html.New(html.HighSecurityConfig()) // 使用预设配置
processor, err := html.New(html.TextOnlyConfig())     // 文本专用预设
processor, err := html.New(html.MarkdownConfig())     // Markdown 预设
defer processor.Close()

// 提取（从字节）
processor.Extract(htmlBytes []byte) (*Result, error)
processor.ExtractText(htmlBytes []byte) (string, error)

// 提取（从文件）
processor.ExtractFromFile(filePath string) (*Result, error)
processor.ExtractTextFromFile(filePath string) (string, error)

// 格式转换（从字节）
processor.ExtractToMarkdown(htmlBytes []byte) (string, error)
processor.ExtractToJSON(htmlBytes []byte) ([]byte, error)

// 格式转换（从文件）
processor.ExtractToMarkdownFromFile(filePath string) (string, error)
processor.ExtractToJSONFromFile(filePath string) ([]byte, error)

// 链接（从字节）
processor.ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error)

// 链接（从文件）
processor.ExtractAllLinksFromFile(filePath string) ([]LinkResource, error)

// 批处理
processor.ExtractBatch(contents [][]byte) ([]*Result, error)
processor.ExtractBatchFiles(paths []string) ([]*Result, error)
processor.ExtractBatchWithContext(ctx context.Context, contents [][]byte) *BatchResult
processor.ExtractBatchFilesWithContext(ctx context.Context, paths []string) *BatchResult

// 监控
processor.GetStatistics() Statistics
processor.ClearCache()
processor.ResetStatistics()
processor.GetAuditLog() []AuditEntry
processor.ClearAuditLog()
```

### 配置函数

```go
// Processor 配置预设
html.DefaultConfig() Config        // 标准配置
html.HighSecurityConfig() Config   // 安全优化配置
html.TextOnlyConfig() Config       // 仅文本（无媒体）
html.MarkdownConfig() Config       // Markdown 图片格式
```

### 默认配置值

**DefaultConfig():**
```go
Config{
    MaxInputSize:       50 * 1024 * 1024, // 50MB
    MaxCacheEntries:    2000,
    CacheTTL:           1 * time.Hour,
    CacheCleanup:       5 * time.Minute,
    WorkerPoolSize:     4,
    EnableSanitization: true,
    MaxDepth:           500,
    ProcessingTimeout:  30 * time.Second,

    // 提取设置
    ExtractArticle:     true,
    PreserveImages:     true,
    PreserveLinks:      true,
    PreserveVideos:     true,
    PreserveAudios:     true,
    ImageFormat:        "none",
    LinkFormat:         "none",
    TableFormat:        "markdown",
}
```

**HighSecurityConfig():**
```go
Config{
    MaxInputSize:       10 * 1024 * 1024, // 10MB - 为安全而减小
    MaxCacheEntries:    500,              // 减小缓存大小
    CacheTTL:           30 * time.Minute, // 更短的 TTL
    CacheCleanup:       1 * time.Minute,  // 更频繁的清理
    WorkerPoolSize:     2,                // 更少的 worker
    EnableSanitization: true,
    MaxDepth:           100,              // 减小深度限制
    ProcessingTimeout:  10 * time.Second, // 更短的超时

    // 提取设置（与 DefaultConfig 相同）
    ExtractArticle:     true,
    PreserveImages:     true,
    PreserveLinks:      true,
    PreserveVideos:     true,
    PreserveAudios:     true,
    ImageFormat:        "none",
    LinkFormat:         "none",
    TableFormat:        "markdown",
}
```

**TextOnlyConfig():**
```go
Config{
    // 继承所有 DefaultConfig 设置，另外：
    PreserveImages:     false,
    PreserveLinks:      false,
    PreserveVideos:     false,
    PreserveAudios:     false,
}
```

**MarkdownConfig():**
```go
Config{
    // 继承所有 DefaultConfig 设置，另外：
    ImageFormat:        "markdown",
}
```

**LinkExtractionOptions (DefaultConfig().LinkExtraction):**
```go
LinkExtractionOptions{
    ResolveRelativeURLs:  true,
    BaseURL:              "",    // 自动检测
    IncludeImages:        true,
    IncludeVideos:        true,
    IncludeAudios:        true,
    IncludeCSS:           true,
    IncludeJS:            true,
    IncludeContentLinks:  true,
    IncludeExternalLinks: true,
    IncludeIcons:         true,
}
```

---

## 结果结构

```go
type Result struct {
    Text           string        `json:"text"`
    Title          string        `json:"title"`
    Images         []ImageInfo   `json:"images,omitempty"`
    Links          []LinkInfo    `json:"links,omitempty"`
    Videos         []VideoInfo   `json:"videos,omitempty"`
    Audios         []AudioInfo   `json:"audios,omitempty"`
    WordCount      int           `json:"word_count"`
    ReadingTime    time.Duration `json:"reading_time_ms"`    // JSON: 毫秒
    ProcessingTime time.Duration `json:"processing_time_ms"` // JSON: 毫秒
}

type ImageInfo struct {
    URL          string `json:"url"`
    Alt          string `json:"alt"`
    Title        string `json:"title"`
    Width        string `json:"width"`
    Height       string `json:"height"`
    IsDecorative bool   `json:"is_decorative"`
    Position     int    `json:"position"`
}

type LinkInfo struct {
    URL        string `json:"url"`
    Text       string `json:"text"`
    Title      string `json:"title"`
    IsExternal bool   `json:"is_external"`
    IsNoFollow bool   `json:"is_nofollow"`
    Position   int    `json:"position"`
}

type VideoInfo struct {
    URL      string `json:"url"`
    Type     string `json:"type"`
    Poster   string `json:"poster"`
    Width    string `json:"width"`
    Height   string `json:"height"`
    Duration string `json:"duration"`
}

type AudioInfo struct {
    URL      string `json:"url"`
    Type     string `json:"type"`
    Duration string `json:"duration"`
}

type LinkResource struct {
    URL   string
    Title string
    Type  string // "css", "js", "image", "video", "audio", "icon", "link"
}

type BatchResult struct {
    Results    []*Result
    Errors     []error
    Success    int
    Failed     int
    Cancelled  int
}

type Statistics struct {
    TotalProcessed     int64
    CacheHits          int64
    CacheMisses        int64
    ErrorCount         int64
    AverageProcessTime time.Duration
}
```

---

## 🔒 安全特性

本库内置安全防护：

### HTML 净化
- **危险标签移除**：`<script>`、`<style>`、`<noscript>`、`<iframe>`、`<embed>`、`<object>`、`<form>`、`<input>`、`<button>`
- **事件处理器移除**：所有 `on*` 属性（onclick、onerror、onload 等）
- **危险协议阻止**：`javascript:`、`vbscript:`、`data:`（安全媒体类型除外）
- **XSS 防护**：全面的净化以防止跨站脚本攻击

### 输入验证
- **大小限制**：可配置的 `MaxInputSize` 防止内存耗尽
- **深度限制**：`MaxDepth` 防止深度嵌套 HTML 导致栈溢出
- **超时保护**：`ProcessingTimeout` 防止格式错误输入导致挂起
- **路径遍历防护**：`ExtractFromFile` 验证文件路径

### Data URL 安全
仅允许安全的媒体类型 data URL：
- **允许**：`data:image/*`、`data:font/*`、`data:application/pdf`
- **阻止**：`data:text/html`、`data:text/javascript`、`data:text/plain`

### 高安全预设
需要更安全的应用请使用 `html.HighSecurityConfig()`：
- 更小的输入大小限制（10MB vs 50MB）
- 更低的深度限制（100 vs 500）
- 更短的超时（10s vs 30s）
- 默认启用审计日志

---

## 🔍 审计日志

本库提供全面的审计日志功能以满足安全合规要求：

### 启用审计日志

```go
// 方法 1：使用 HighSecurityConfig（默认启用审计）
processor, _ := html.New(html.HighSecurityConfig())

// 方法 2：自定义配置并启用审计
config := html.DefaultConfig()
config.Audit = html.AuditConfig{
    Enabled:            true,
    LogBlockedTags:     true,
    LogBlockedAttrs:    true,
    LogBlockedURLs:     true,
    LogInputViolations: true,
    LogDepthViolations: true,
    LogTimeouts:        true,
    LogEncodingIssues:  true,
    LogPathTraversal:   true,
}
processor, _ := html.New(config)
```

### 获取审计日志

```go
processor, _ := html.New(html.HighSecurityConfig())
defer processor.Close()

// 处理内容
processor.Extract(htmlBytes)

// 获取审计条目
entries := processor.GetAuditLog()
for _, entry := range entries {
    fmt.Printf("[%s] %s: %s\n", entry.Level, entry.EventType, entry.Message)
}

// 清除审计日志
processor.ClearAuditLog()
```

### 审计条目结构

```go
type AuditEntry struct {
    Timestamp time.Time      `json:"timestamp"`
    EventType AuditEventType `json:"event_type"`
    Level     AuditLevel     `json:"level"`
    Message   string         `json:"message"`
    Tag       string         `json:"tag,omitempty"`
    Attribute string         `json:"attribute,omitempty"`
    URL       string         `json:"url,omitempty"`
    InputSize int            `json:"input_size,omitempty"`
    MaxSize   int            `json:"max_size,omitempty"`
    Depth     int            `json:"depth,omitempty"`
    MaxDepth  int            `json:"max_depth,omitempty"`
    Path      string         `json:"path,omitempty"`
    RawValue  string         `json:"raw_value,omitempty"`
}
```

### 自定义审计输出目标

```go
import "github.com/cybergodev/html"

// 将审计日志写入文件
file, _ := os.Create("audit.log")
fileSink := html.NewWriterAuditSink(file)

// 仅过滤关键事件
filteredSink := html.NewLevelFilteredSink(fileSink, html.AuditLevelCritical)

// 在配置中使用自定义输出目标
config := html.DefaultConfig()
config.Audit = html.AuditConfig{
    Enabled: true,
    Sink:    filteredSink,
}
processor, _ := html.New(config)
```

### 内置审计输出目标

| 输出目标 | 描述 |
|---------|------|
| `NewLoggerAuditSink()` | 写入 stderr，带 `[AUDIT]` 前缀 |
| `NewLoggerAuditSinkWithWriter(w)` | 写入自定义 io.Writer |
| `NewWriterAuditSink(w)` | 以 JSON 行格式写入 io.Writer |
| `NewChannelAuditSink(bufferSize)` | 发送到通道供外部处理 |
| `NewMultiSink(sinks...)` | 组合多个输出目标 |
| `NewFilteredSink(sink, filter)` | 写入前过滤条目 |
| `NewLevelFilteredSink(sink, level)` | 仅输出指定级别及以上的条目 |

---

## 示例代码

完整的可运行示例请参见 [examples/](examples) 目录：

| 示例 | 描述 |
|------|------|
| [01_quick_start.go](examples/01_quick_start.go) | 快速入门指南 |
| [02_content_extraction.go](examples/02_content_extraction.go) | 内容提取选项与输出格式 |
| [03_links_media.go](examples/03_links_media.go) | 链接与媒体提取 |
| [04_configuration.go](examples/04_configuration.go) | 配置与性能调优 |
| [05_http_integration.go](examples/05_http_integration.go) | HTTP 集成模式 |
| [06_advanced_usage.go](examples/06_advanced_usage.go) | 自定义评分器、审计日志、安全配置 |
| [07_error_handling.go](examples/07_error_handling.go) | 错误处理模式 |
| [08_real_world.go](examples/08_real_world.go) | 实际应用案例 |

---

## 兼容性

本库是 `golang.org/x/net/html` 的**直接替代品**：

```go
// 只需更改导入
- import "golang.org/x/net/html"
+ import "github.com/cybergodev/html"

// 所有现有代码继续工作
doc, err := html.Parse(reader)
html.Render(writer, doc)
escaped := html.EscapeString("<script>")
```

本库重新导出了常用的类型、常量和函数：
- **类型**：`Node`、`NodeType`、`Token`、`Attribute`、`Tokenizer`、`ParseOption`
- **常量**：所有 `NodeType` 和 `TokenType` 常量（`ErrorNode`、`TextNode`、`DocumentNode`、`ElementNode` 等）
- **函数**：`Parse`、`ParseFragment`、`Render`、`EscapeString`、`UnescapeString`、`NewTokenizer`、`NewTokenizerFragment`

---

## 线程安全

`Processor` 可安全并发使用：

```go
processor, _ := html.New()
defer processor.Close()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.Extract(htmlBytes)
    }()
}
wg.Wait()
```

---

## 🤝 贡献

欢迎贡献！提交 PR 前请阅读贡献指南。

## 📄 许可证

MIT 许可证 - 详情见 [LICENSE](LICENSE) 文件。

---

**用心为 Go 社区打造**

如果这个项目对你有帮助，请给个 Star！
