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
htmlContent, _ := html.ExtractText(`
    <html>
        <nav>导航菜单</nav>
        <article><h1>Hello World</h1><p>内容在这里...</p></article>
        <footer>页脚</footer>
    </html>
`)
fmt.Println(htmlContent) // "Hello World\n内容在这里..."
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
text, _ := html.ExtractText(htmlContent)

// 提取所有内容
result, _ := html.Extract(htmlContent)
fmt.Println(result.Title)     // Hello World
fmt.Println(result.Text)      // 内容在这里...
fmt.Println(result.WordCount) // 4

// 提取所有资源链接
links, _ := html.ExtractAllLinks(htmlContent)

// 转换格式
markdown, _ := html.ExtractToMarkdown(htmlContent)
jsonData, _ := html.ExtractToJSON(htmlContent)
```

**适用场景:** 简单脚本、一次性任务、快速原型开发

---

### 基础 Processor 用法

需要更多控制？创建一个处理器:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 使用默认配置提取
result, _ := processor.ExtractWithDefaults(htmlContent)

// 从文件提取
result, _ = processor.ExtractFromFile("page.html", html.DefaultExtractConfig())

// 批处理
htmlContents := []string{html1, html2, html3}
results, _ := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())
```

**适用场景:** 多次提取、处理多个文件、网页爬虫

---

### 自定义配置

精确控制提取内容:

```go
config := html.ExtractConfig{
    ExtractArticle:    true,       // 自动检测主要内容
    PreserveImages:    true,       // 提取图片元数据
    PreserveLinks:     true,       // 提取链接元数据
    PreserveVideos:    false,      // 跳过视频
    PreserveAudios:    false,      // 跳过音频
    InlineImageFormat: "none",     // 选项: "none", "placeholder", "markdown", "html"
    TableFormat:       "markdown", // 选项: "markdown", "html"
}

processor := html.NewWithDefaults()
defer processor.Close()

result, _ := processor.Extract(htmlContent, config)
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
    MaxDepth:           50,    // 防止深层嵌套攻击
}

processor, _ := html.New(config)
defer processor.Close()
```

#### 链接提取

```go
// 提取所有资源链接
links, _ := html.ExtractAllLinks(htmlContent)

// 按类型分组
byType := html.GroupLinksByType(links)
cssLinks := byType["css"]
jsLinks := byType["js"]
images := byType["image"]

// 高级配置
processor := html.NewWithDefaults()
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
links, _ = processor.ExtractAllLinks(htmlContent, linkConfig)
```

#### 缓存与统计

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 自动启用缓存
result1, _ := processor.ExtractWithDefaults(htmlContent)
result2, _ := processor.ExtractWithDefaults(htmlContent) // 缓存命中!

// 检查性能
stats := processor.GetStatistics()
fmt.Printf("缓存命中: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

// 需要时清除缓存
processor.ClearCache()
```

**适用场景:** 生产应用、性能优化、特定用例

---

## 📖 常用示例

常见问题的复制粘贴解决方案:

### 提取文章文本（纯净版）

```go
text, _ := html.ExtractText(htmlContent)
// 返回不含导航/广告的纯净文本
```

### 提取并保留图片

```go
result, _ := html.Extract(htmlContent)
for _, img := range result.Images {
    fmt.Printf("图片: %s (alt: %s)\n", img.URL, img.Alt)
}
```

### 转换为 Markdown

```go
markdown, _ := html.ExtractToMarkdown(htmlContent)
// 图片变成: ![alt](url)
```

### 提取所有链接

```go
links, _ := html.ExtractAllLinks(htmlContent)
for _, link := range links {
    fmt.Printf("%s: %s\n", link.Type, link.URL)
}
```

### 获取阅读时间

```go
result, _ := html.Extract(htmlContent)
minutes := result.ReadingTime.Minutes()
fmt.Printf("阅读时间: %.1f 分钟", minutes)
```

### 批处理文件

```go
processor := html.NewWithDefaults()
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, _ := processor.ExtractBatchFiles(files, html.DefaultExtractConfig())
```

---

## 🔧 API 快速参考

### 包级函数

```go
// 内容提取
html.Extract(htmlContent string, configs ...ExtractConfig) (*Result, error)
html.ExtractText(htmlContent string) (string, error)
html.ExtractFromFile(path string, configs ...ExtractConfig) (*Result, error)

// 格式转换
html.ExtractToMarkdown(htmlContent string) (string, error)
html.ExtractToJSON(htmlContent string) ([]byte, error)

// 链接提取
html.ExtractAllLinks(htmlContent string, configs ...LinkExtractionConfig) ([]LinkResource, error)
html.GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### Processor 方法

```go
// 创建 (包级函数)
processor := html.NewWithDefaults()
// 或
processor, err := html.New(config)
defer processor.Close()

// 内容提取
processor.Extract(htmlContent string, config ExtractConfig) (*Result, error)
processor.ExtractWithDefaults(htmlContent string) (*Result, error)
processor.ExtractFromFile(path string, config ExtractConfig) (*Result, error)

// 批处理
processor.ExtractBatch(contents []string, config ExtractConfig) ([]*Result, error)
processor.ExtractBatchFiles(paths []string, config ExtractConfig) ([]*Result, error)

// 链接提取
processor.ExtractAllLinks(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error)

// 监控
processor.GetStatistics() Statistics
processor.ClearCache()
```

### 配置函数

```go
// Processor 配置
html.DefaultConfig()            Config

// 内容提取配置
html.DefaultExtractConfig()           ExtractConfig

// 链接提取配置
html.DefaultLinkExtractionConfig()           LinkExtractionConfig
```

---

## Result 结构体

```go
type Result struct {
    Text           string        // 纯文本内容
    Title          string        // 页面/文章标题
    Images         []ImageInfo   // 图片元数据
    Links          []LinkInfo    // 链接元数据
    Videos         []VideoInfo   // 视频元数据
    Audios         []AudioInfo   // 音频元数据
    WordCount      int           // 总字数
    ReadingTime    time.Duration // 预估阅读时间 (JSON: reading_time_ms，单位毫秒)
    ProcessingTime time.Duration // 处理耗时 (JSON: processing_time_ms，单位毫秒)
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
    Poster   string  // 封面图片 URL
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

---

## 示例

查看 [examples/](examples) 目录获取完整可运行的代码:

| 示例                                                           | 描述                          |
|---------------------------------------------------------------|--------------------------------|
| [01_quick_start.go](examples/01_quick_start.go)               | 一行代码快速入门                |
| [02_content_extraction.go](examples/02_content_extraction.go) | 内容提取基础                   |
| [03_media_and_links.go](examples/03_media_and_links.go)       | 媒体和链接提取                 |
| [04_advanced_usage.go](examples/04_advanced_usage.go)         | 高级功能和批处理               |
| [05_output_formats.go](examples/05_output_formats.go)         | JSON 和 Markdown 输出格式      |
| [06_error_handling.go](examples/06_error_handling.go)         | 错误处理模式                   |
| [07_real_world.go](examples/07_real_world.go)                 | 真实世界用例                   |
| [08_compatibility.go](examples/08_compatibility.go)           | golang.org/x/net/html 兼容性   |

---

## 兼容性

本库是 **golang.org/x/net/html 的直接替代品**:

```go
// 只需修改导入路径
- import "golang.org/x/net/html"
+ import "github.com/cybergodev/html"

// 所有现有代码正常工作
doc, err := html.Parse(reader)
html.Render(writer, doc)
escaped := html.EscapeString("<script>")
```

本库重新导出了 `golang.org/x/net/html` 的所有常用类型、常量和函数:
- **类型**: `Node`, `NodeType`, `Token`, `Attribute`, `Tokenizer`, `ParseOption`
- **常量**: 所有 `NodeType` 和 `TokenType` 常量
- **函数**: `Parse`, `ParseFragment`, `ParseWithOptions`, `ParseFragmentWithOptions`, `Render`, `EscapeString`, `UnescapeString`, `NewTokenizer`, `NewTokenizerFragment`, `ParseOptionEnableScripting`

---

## 线程安全

`Processor` 支持并发使用:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 可安全地在多个 goroutine 中使用
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.ExtractWithDefaults(htmlContent)
    }()
}
wg.Wait()
```

---

## 🤝 贡献

欢迎贡献代码、报告问题和提出建议！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**用心为 Go 社区打造** ❤️ | 如果这个项目对你有帮助，请给它一个 ⭐️ Star！
