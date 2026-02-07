# HTML 库

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/html)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/html)


**一个用于智能 HTML 内容提取的 Go 库。** 与 `golang.org/x/net/html` 兼容 — 可作为直接替代品使用，并获得增强的内容提取功能。

#### **[📖 English Documentation](README.md)** - 用户指南

## ✨ 核心功能

### 🎯 内容提取
- **文章检测**: 使用评分算法识别主要内容（文本密度、链接密度、语义标签）
- **智能文本提取**: 保留结构、处理换行、计算字数和阅读时间
- **媒体提取**: 图片、视频、音频及其元数据（URL、尺寸、替代文本、类型检测）
- **链接分析**: 外部/内部链接检测、nofollow 属性、锚文本提取

### ⚡ 性能
- **内容寻址缓存**: 基于 SHA256 的键、TTL 和 LRU 淘汰策略
- **批处理**: 可配置工作池的并行提取
- **线程安全**: 支持并发使用，无需外部同步
- **资源限制**: 可配置的输入大小、嵌套深度和超时保护

### 📖 使用场景
- **新闻聚合器**: 从新闻网站提取文章内容
- **网页爬虫**: 从 HTML 页面获取结构化数据
- **内容管理**: 将 HTML 转换为 Markdown 或其他格式
- **搜索引擎**: 索引主要内容，排除导航/广告
- **数据分析**: 大规模提取和分析网页内容
- **RSS/Feed 生成器**: 从 HTML 内容生成订阅源
- **文档工具**: 将 HTML 文档转换为其他格式

---

## 📦 安装

```bash
go get github.com/cybergodev/html
```

---

## ⚡ 5 分钟快速入门

```go
import "github.com/cybergodev/html"

// 从 HTML 提取纯文本
htmlBytes := []byte(`
    <html>
        <nav>导航菜单</nav>
        <article><h1>Hello World</h1><p>内容在这里...</p></article>
        <footer>页脚</footer>
    </html>
`)
text, _ := html.ExtractText(htmlBytes)
fmt.Println(text) // "Hello World\n内容在这里..."
```

**就这么简单！** 库会自动:
- 移除导航、页脚、广告
- 提取主要内容
- 清理空白字符

---

## 🚀 快速指南

### 一行代码函数

只想快速完成工作？使用这些包级函数:

```go
// 仅提取文本
text, _ := html.ExtractText(htmlBytes)

// 提取所有内容
result, _ := html.Extract(htmlBytes)
fmt.Println(result.Title)     // Hello World
fmt.Println(result.Text)      // 内容在这里...
fmt.Println(result.WordCount) // 4

// 提取所有资源链接
links, _ := html.ExtractAllLinks(htmlBytes)

// 转换格式
markdown, _ := html.ExtractToMarkdown(htmlBytes)
jsonData, _ := html.ExtractToJSON(htmlBytes)
```

**适用场景:** 简单脚本、一次性任务、快速原型开发

---

### 基础 Processor 用法

需要更多控制？创建一个处理器:

```go
// 使用默认配置创建处理器
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

// 使用默认配置提取
result, _ := processor.ExtractWithDefaults(htmlBytes)

// 从文件提取
result, _ = processor.ExtractFromFile("page.html", html.DefaultExtractConfig())

// 批处理
htmlContents := [][]byte{html1, html2, html3}
results, _ := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())
```

**适用场景:** 多次提取、处理多个文件、网页爬虫

---

### 自定义配置

微调提取内容:

```go
config := html.ExtractConfig{
    ExtractArticle:    true,       // 自动检测主要内容
    PreserveImages:    true,       // 提取图片元数据
    PreserveLinks:     true,       // 提取链接元数据
    PreserveVideos:    false,      // 跳过视频
    PreserveAudios:    false,      // 跳过音频
    InlineImageFormat: "none",     // 选项: "none", "placeholder", "markdown", "html"
    TableFormat:       "markdown", // 选项: "markdown", "html"
    Encoding:          "",         // 从 meta 标签自动检测，或指定: "utf-8", "windows-1252" 等
}

processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

result, _ := processor.Extract(htmlBytes, config)
```

**适用场景:** 特定提取需求、格式转换、自定义输出

---

### 高级功能

#### 自定义 Processor 配置

```go
config := html.Config{
    MaxInputSize:       10 * 1024 * 1024, // 10MB 限制
    ProcessingTimeout:  30 * time.Second,
    MaxCacheEntries:    500,
    CacheTTL:           30 * time.Minute,
    WorkerPoolSize:     8,
    EnableSanitization: true,  // 移除 <script>, <style> 标签
    MaxDepth:           50,    // 防止深度嵌套攻击
}

processor, _ := html.New(config)
defer processor.Close()
```

#### 链接提取

```go
// 提取所有资源链接
links, _ := html.ExtractAllLinks(htmlBytes)

// 按类型分组
byType := html.GroupLinksByType(links)
cssLinks := byType["css"]
jsLinks := byType["js"]
images := byType["image"]

// 高级配置
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
linkConfig := html.LinkExtractionConfig{
    BaseURL:               "https://example.com",
    ResolveRelativeURLs:   true,
    IncludeImages:         true,
    IncludeVideos:         true,
    IncludeAudios:         true,
    IncludeCSS:            true,
    IncludeJS:             true,
    IncludeContentLinks:   true,
    IncludeExternalLinks:  true,
    IncludeIcons:          true,
}
links, _ = processor.ExtractAllLinks(htmlBytes, linkConfig)
```

#### 缓存与统计

```go
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

// 自动启用缓存
result1, _ := processor.ExtractWithDefaults(htmlBytes)
result2, _ := processor.ExtractWithDefaults(htmlBytes) // 缓存命中！

// 检查性能
stats := processor.GetStatistics()
fmt.Printf("缓存命中: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

// 清除缓存（保留统计信息）
processor.ClearCache()

// 重置统计（保留缓存条目）
processor.ResetStatistics()
```

**适用场景:** 生产应用、性能优化、特定用例

---

## 📖 常用示例

可直接复制使用的解决方案:

### 提取文章文本（纯净版）

```go
text, _ := html.ExtractText(htmlBytes)
// 返回不含导航/广告的纯净文本
```

### 提取包含图片的内容

```go
result, _ := html.Extract(htmlBytes)
for _, img := range result.Images {
    fmt.Printf("图片: %s (alt: %s)\n", img.URL, img.Alt)
}
```

### 转换为 Markdown

```go
markdown, _ := html.ExtractToMarkdown(htmlBytes)
// 图片变成: ![alt](url)
```

### 提取所有链接

```go
links, _ := html.ExtractAllLinks(htmlBytes)
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

### 批处理文件

```go
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, _ := processor.ExtractBatchFiles(files, html.DefaultExtractConfig())
```

---

## 🔧 API 快速参考

### 包级函数

```go
// 提取
html.Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error)
html.ExtractText(htmlBytes []byte) (string, error)
html.ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error)

// 格式转换
html.ExtractToMarkdown(htmlBytes []byte) (string, error)
html.ExtractToJSON(htmlBytes []byte) ([]byte, error)

// 链接
html.ExtractAllLinks(htmlBytes []byte, configs ...LinkExtractionConfig) ([]LinkResource, error)
html.GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### Processor 方法

```go
// 创建
processor, err := html.New()
// 或使用自定义配置:
processor, err := html.New(config)
defer processor.Close()

// 提取
processor.Extract(htmlBytes []byte, config ExtractConfig) (*Result, error)
processor.ExtractWithDefaults(htmlBytes []byte) (*Result, error)
processor.ExtractFromFile(filePath string, config ExtractConfig) (*Result, error)

// 批处理
processor.ExtractBatch(contents [][]byte, config ExtractConfig) ([]*Result, error)
processor.ExtractBatchFiles(paths []string, config ExtractConfig) ([]*Result, error)

// 链接
processor.ExtractAllLinks(htmlBytes []byte, config LinkExtractionConfig) ([]LinkResource, error)

// 监控
processor.GetStatistics() Statistics
processor.ClearCache()
processor.ResetStatistics()
```

### 配置函数

```go
// Processor 配置
html.DefaultConfig()            Config

// 提取配置
html.DefaultExtractConfig()           ExtractConfig

// 链接提取配置
html.DefaultLinkExtractionConfig()           LinkExtractionConfig
```

**`DefaultConfig()` 的默认值:**
```go
Config{
    MaxInputSize:       50 * 1024 * 1024, // 50MB
    MaxCacheEntries:    2000,
    CacheTTL:           1 * time.Hour,
    WorkerPoolSize:     4,
    EnableSanitization: true,
    MaxDepth:           500,
    ProcessingTimeout:  30 * time.Second,
}
```

**`DefaultExtractConfig()` 的默认值:**
```go
ExtractConfig{
    ExtractArticle:    true,
    PreserveImages:    true,
    PreserveLinks:     true,
    PreserveVideos:    true,
    PreserveAudios:    true,
    InlineImageFormat: "none",
    TableFormat:       "markdown",
    Encoding:          "", // 自动检测
}
```

**`DefaultLinkExtractionConfig()` 的默认值:**
```go
LinkExtractionConfig{
    ResolveRelativeURLs:  true,  // 将相对 URL 转换为绝对 URL
    BaseURL:              "",    // 解析的基础 URL（空 = 自动检测）
    IncludeImages:        true,  // 提取图片链接
    IncludeVideos:        true,  // 提取视频链接
    IncludeAudios:        true,  // 提取音频链接
    IncludeCSS:           true,  // 提取 CSS 链接
    IncludeJS:            true,  // 提取 JavaScript 链接
    IncludeContentLinks:  true,  // 提取内容链接
    IncludeExternalLinks: true,  // 提取外部域名链接
    IncludeIcons:         true,  // 提取 favicon/icon 链接
}
```

---

## 结果结构

```go
type Result struct {
    Text           string        // 纯净文本内容
    Title          string        // 页面/文章标题
    Images         []ImageInfo   // 图片元数据
    Links          []LinkInfo    // 链接元数据
    Videos         []VideoInfo   // 视频元数据
    Audios         []AudioInfo   // 音频元数据
    WordCount      int           // 总字数
    ReadingTime    time.Duration // 预估阅读时间 (JSON: reading_time_ms 毫秒)
    ProcessingTime time.Duration // 耗时 (JSON: processing_time_ms 毫秒)
}

type ImageInfo struct {
    URL          string  // 图片 URL
    Alt          string  // 替代文本
    Title        string  // 标题属性
    Width        string  // 宽度属性
    Height       string  // 高度属性
    IsDecorative bool    // 无替代文本
    Position     int     // 在文档中的位置
}

type LinkInfo struct {
    URL        string  // 链接 URL
    Text       string  // 锚文本
    Title      string  // 标题属性
    IsExternal bool    // 外部域名
    IsNoFollow bool    // rel="nofollow"
}

type VideoInfo struct {
    URL      string  // 视频 URL
    Type     string  // MIME 类型或 "embed"
    Poster   string  // 海报图片 URL
    Width    string  // 宽度属性
    Height   string  // 高度属性
    Duration string  // 时长属性
}

type AudioInfo struct {
    URL      string  // 音频 URL
    Type     string  // MIME 类型
    Duration string  // 时长属性
}

type LinkResource struct {
    URL   string  // 资源 URL
    Title string  // 资源标题
    Type  string  // 资源类型: css, js, image, video, audio, icon, link 或 media
}
```

### 统计结构

```go
type Statistics struct {
    TotalProcessed     int64         // 执行的提取总数
    CacheHits          int64         // 缓存命中次数
    CacheMisses        int64         // 缓存未命中次数
    ErrorCount         int64         // 遇到的错误数
    AverageProcessTime time.Duration // 平均每次提取的处理时间
}
```

---

## 🔒 安全特性

库包含内置的安全保护机制:

### HTML 清理
- **危险标签移除**: `<script>`, `<style>`, `<noscript>`, `<iframe>`, `<embed>`, `<object>`, `<form>`, `<input>`, `<button>`
- **事件处理器移除**: 所有 `on*` 属性（onclick, onerror, onload 等）
- **危险协议阻止**: `javascript:`, `vbscript:`, `data:`（安全媒体类型除外）
- **XSS 预防**: 全面的清理以防止跨站脚本攻击

### 输入验证
- **大小限制**: 可配置的 `MaxInputSize` 防止内存耗尽
- **深度限制**: `MaxDepth` 防止深度嵌套 HTML 导致的栈溢出
- **超时保护**: `ProcessingTimeout` 防止畸形输入导致挂起
- **路径遍历保护**: `ExtractFromFile` 验证文件路径以防止目录遍历攻击

### 数据 URL 安全
仅允许安全的媒体类型数据 URL:
- **允许**: `data:image/*`, `data:font/*`, `data:application/pdf`
- **阻止**: `data:text/html`, `data:text/javascript`, `data:text/plain`

---

## 性能基准

基于 `benchmark_test.go`:

| 操作 | 性能 | 说明 |
|------|------|------|
| 文本提取 | ~500ns 每个 HTML 文档 | 快速文本提取 |
| 链接提取 | ~2μs 每个 HTML 文档 | 包含元数据提取 |
| 完整提取 | ~5μs 每个 HTML 文档 | 启用所有功能 |
| 缓存命中 | ~100ns | 缓存内容近即时 |

**缓存优势:**
- **SHA256 键**: 内容寻址缓存
- **TTL 支持**: 可配置的缓存过期
- **LRU 淘汰**: 使用双向链表的自动缓存管理
- **线程安全**: 并发访问无需外部锁

---

查看 [examples/](examples) 目录获取完整可运行的代码:

| 示例                                                           | 描述                          |
|----------------------------------------------------------------|-----------------------------------|
| [01_quick_start.go](examples/01_quick_start.go)               | 快速入门单行代码                |
| [02_content_extraction.go](examples/02_content_extraction.go) | 内容提取基础                    |
| [03_media_and_links.go](examples/03_media_and_links.go)       | 媒体和链接提取                  |
| [04_advanced_usage.go](examples/04_advanced_usage.go)         | 高级功能和批处理                |
| [05_output_formats.go](examples/05_output_formats.go)         | JSON 和 Markdown 输出格式       |
| [06_error_handling.go](examples/06_error_handling.go)         | 错误处理模式                    |
| [07_real_world.go](examples/07_real_world.go)                 | 实际用例                        |
| [08_compatibility.go](examples/08_compatibility.go)           | golang.org/x/net/html 兼容性   |

---

## 兼容性

本库是 `golang.org/x/net/html` 的**直接替代品**:

```go
// 只需修改导入
- import "golang.org/x/net/html"
+ import "github.com/cybergodev/html"

// 所有现有代码都能工作
doc, err := html.Parse(reader)
html.Render(writer, doc)
escaped := html.EscapeString("<script>")
```

库重新导出 `golang.org/x/net/html` 的所有常用类型、常量和函数:
- **类型**: `Node`, `NodeType`, `Token`, `Attribute`, `Tokenizer`, `ParseOption`
- **常量**: 所有 `NodeType` 和 `TokenType` 常量
- **函数**: `Parse`, `ParseFragment`, `ParseWithOptions`, `ParseFragmentWithOptions`, `Render`, `EscapeString`, `UnescapeString`, `NewTokenizer`, `NewTokenizerFragment`, `ParseOptionEnableScripting`

---

## 线程安全

`Processor` 可安全地并发使用:

```go
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

// 可从多个 goroutine 安全使用
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.ExtractWithDefaults(htmlBytes)
    }()
}
wg.Wait()
```

---

## 🤝 贡献

欢迎贡献代码、报告问题和提出建议！

## 📄 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

**为 Go 社区精心打造** ❤️ | 如果这个项目对你有帮助，请给它一个 ⭐️ Star！

---

## 错误处理

库为不同的失败场景提供特定的错误类型:

```go
var (
    ErrInputTooLarge     = errors.New("html: input size exceeds maximum")
    ErrInvalidHTML       = errors.New("html: invalid HTML content")
    ErrInvalidConfig     = errors.New("html: invalid configuration")
    ErrProcessorClosed   = errors.New("html: processor closed")
    ErrFileNotFound      = errors.New("html: file not found")
    ErrInvalidFilePath   = errors.New("html: invalid file path")
    ErrMaxDepthExceeded  = errors.New("html: max depth exceeded")
    ErrProcessingTimeout = errors.New("html: processing timeout")
)
```

### 错误处理最佳实践

```go
result, err := html.Extract(htmlBytes)
if err != nil {
    if errors.Is(err, html.ErrInputTooLarge) {
        // 处理超大输入
    } else if errors.Is(err, html.ErrInvalidHTML) {
        // 处理畸形 HTML
    } else if errors.Is(err, html.ErrProcessorClosed) {
        // 处理已关闭的处理器
    } else {
        // 处理其他错误
        log.Printf("提取失败: %v", err)
    }
    return
}
```

---

## 字符编码支持

库可自动检测并转换 15+ 种字符编码的内容:

### 支持的编码

**Unicode:**
- UTF-8, UTF-16 LE, UTF-16 BE

**西欧:**
- Windows-1252, ISO-8859-1 至 ISO-8859-16

**东亚:**
- GBK, Big5, Shift_JIS, EUC-JP, ISO-2022-JP, EUC-KR

### 编码检测

库使用三层检测策略:
1. **BOM 检测**: UTF-8/UTF-16 的字节顺序标记
2. **Meta 标签检测**: HTML `<meta charset>` 和 `http-equiv` 头
3. **智能检测**: 基于统计分析的置信度评分

### 手动指定编码

```go
config := html.ExtractConfig{
    Encoding: "windows-1252", // 强制指定编码
}
result, _ := html.Extract(htmlBytes, config)
```

---

## 最新改进

### 性能与质量改进 (2026-02-07)

- ✅ **修复 LRU 缓存 Bug**: 实现正确的双向链表淘汰策略
- ✅ **优化字符串操作**: 减少冗余的 ToLower 转换
- ✅ **延迟正则编译**: 使用 sync.Once 加快启动
- ✅ **改进统计功能**: 添加 ResetStatistics() 方法
- ✅ **统一 URL 验证**: 验证逻辑的单一来源

### 测试套件优化 (2026-02-07)

- ✅ **87.1% 覆盖率**: 从 81.7% 提升（+6.6%）
- ✅ **消除冗余**: 删除重复测试
- ✅ **更好组织**: 整合和结构化测试
- ✅ **完善文档**: 创建测试策略指南

### 文档改进 (2026-02-07)

- ✅ **修正 API 签名**: 所有函数参数类型从 `string` 更正为 `[]byte`
- ✅ **补充遗漏方法**: 添加 `ResetStatistics()` 文档
- ✅ **验证代码示例**: 创建自动化测试套件
- ✅ **100% 准确率**: 所有文档经测试验证

---
