# HTML 库

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/json)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/json)


**一个用于 HTML 内容智能提取的 Go 库。** 与 `golang.org/x/net/html` 兼容 — 可作为直接替代品使用，并提供增强的内容提取功能。

#### **[English Documentation](README.md)** - 英文文档

## 功能特性

### 内容提取
- **文章检测**：使用评分算法识别主要内容（文本密度、链接密度、语义标签）
- **智能文本提取**：保留结构、处理换行、计算字数和阅读时间
- **媒体提取**：图片、视频、音频及其元数据（URL、尺寸、alt 文本、类型检测）
- **链接分析**：外部/内部链接检测、nofollow 属性、锚文本提取

### 性能
- **内容寻址缓存**：基于 SHA256 的缓存键，支持 TTL 和 LRU 淘汰
- **批量处理**：使用可配置的工作池并行提取
- **线程安全**：支持并发使用，无需外部同步
- **资源限制**：可配置的输入大小、嵌套深度和超时保护

### 应用场景
- 📰 **新闻聚合器**：从新闻网站提取文章内容
- 🤖 **网络爬虫**：从 HTML 页面获取结构化数据
- 📝 **内容管理**：将 HTML 转换为 Markdown 或其他格式
- 🔍 **搜索引擎**：索引主要内容，排除导航/广告
- 📊 **数据分析**：大规模提取和分析网页内容
- 📱 **RSS/Feed 生成器**：从 HTML 内容创建订阅源
- 🎓 **文档工具**：将 HTML 文档转换为其他格式

---

## 安装

```bash
go get github.com/cybergodev/html
```

---

## 5 分钟快速开始

```go
import "github.com/cybergodev/html"

// 从 HTML 中提取纯文本
text, _ := html.ExtractText(`
    <html>
        <nav>导航栏</nav>
        <article><h1>Hello World</h1><p>这里是内容...</p></article>
        <footer>页脚</footer>
    </html>
`)
fmt.Println(text) // "Hello World\n这里是内容..."
```

**就这么简单！** 库会自动：
- 移除导航栏、页脚、广告
- 提取主要内容
- 清理空白字符

---

## 快速指南

### 一行代码

只想完成任务？使用这些包级函数：

```go
// 只提取文本
text, _ := html.ExtractText(htmlContent)

// 提取所有内容
result, _ := html.Extract(htmlContent)
fmt.Println(result.Title)     // Hello World
fmt.Println(result.Text)      // 清理后的文本
fmt.Println(result.WordCount) // 5

// 只提取特定元素
title, _ := html.ExtractTitle(htmlContent)
images, _ := html.ExtractImages(htmlContent)
links, _ := html.ExtractLinks(htmlContent)

// 格式转换
markdown, _ := html.ExtractToMarkdown(htmlContent)
jsonData, _ := html.ExtractToJSON(htmlContent)

// 内容分析
wordCount, _ := html.GetWordCount(htmlContent)
readingTime, _ := html.GetReadingTime(htmlContent)
summary, _ := html.Summarize(htmlContent, 50) // 最多 50 个单词
```

**适用场景：** 简单脚本、一次性任务、快速原型

---

### 基础处理器用法

需要更多控制，创建处理器：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 使用默认配置提取
result, _ := processor.ExtractWithDefaults(htmlContent)

// 从文件提取
result, _ = processor.ExtractFromFile("page.html", html.DefaultExtractConfig())

// 批量处理
htmlContents := []string{html1, html2, html3}
results, _ := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())
```

**适用场景：** 多次提取、处理多个文件、网络爬虫

---

### 自定义配置

精细控制提取内容：

```go
config := html.ExtractConfig{
    ExtractArticle:    true,   // 自动检测主要内容
    PreserveImages:    true,   // 提取图片元数据
    PreserveLinks:     true,   // 提取链接元数据
    PreserveVideos:    false,  // 跳过视频
    PreserveAudios:    false,  // 跳过音频
    InlineImageFormat: "none", // 选项："none"、"placeholder"、"markdown"、"html"
}

processor := html.NewWithDefaults()
defer processor.Close()

result, _ := processor.Extract(htmlContent, config)
```

**适用场景：** 特定提取需求、格式转换、自定义输出

---

### 高级功能

#### 自定义处理器配置

```go
config := html.Config{
    MaxInputSize:       10 * 1024 * 1024, // 10MB 限制
    ProcessingTimeout:  30 * time.Second,
    MaxCacheEntries:    500,
    CacheTTL:           30 * time.Minute,
    WorkerPoolSize:     8,
    EnableSanitization: true,  // 移除 <script>、<style> 标签
    MaxDepth:           50,    // 防止深度嵌套攻击
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
    BaseURL:              "https://example.com",
    ResolveRelativeURLs:  true,
    IncludeImages:        true,
    IncludeVideos:        true,
    IncludeCSS:           true,
    IncludeJS:            true,
}
links, _ = processor.ExtractAllLinks(htmlContent, linkConfig)
```

#### 缓存与统计

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 自动缓存已启用
result1, _ := processor.ExtractWithDefaults(htmlContent)
result2, _ := processor.ExtractWithDefaults(htmlContent) // 缓存命中！

// 检查性能
stats := processor.GetStatistics()
fmt.Printf("缓存命中: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

// 需要时清空缓存
processor.ClearCache()
```

#### 配置预设

```go
processor := html.NewWithDefaults()
defer processor.Close()

// RSS 订阅源生成
result, _ := processor.Extract(htmlContent, html.ConfigForRSS())

// 摘要生成（仅文本）
result, _ = processor.Extract(htmlContent, html.ConfigForSummary())

// 搜索索引（所有元数据）
result, _ = processor.Extract(htmlContent, html.ConfigForSearchIndex())

// Markdown 输出
result, _ = processor.Extract(htmlContent, html.ConfigForMarkdown())
```

**适用场景：** 生产应用、性能优化、特定用例

---

## 常用示例

常见任务的复制粘贴解决方案：

### 提取文章文本（清理）

```go
text, _ := html.ExtractText(htmlContent)
// 返回清理后的文本，不包含导航/广告
```

### 提取并包含图片

```go
result, _ := html.Extract(htmlContent)
for _, img := range result.Images {
    fmt.Printf("图片: %s (alt: %s)\n", img.URL, img.Alt)
}
```

### 转换为 Markdown

```go
markdown, _ := html.ExtractToMarkdown(htmlContent)
// 图片变为：![alt](url)
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
minutes, _ := html.GetReadingTime(htmlContent)
fmt.Printf("阅读时间: %.1f 分钟", minutes)
```

### 批量处理文件

```go
processor := html.NewWithDefaults()
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, _ := processor.ExtractBatchFiles(files, html.DefaultExtractConfig())
```

### 创建 RSS 订阅源内容

```go
processor := html.NewWithDefaults()
defer processor.Close()

result, _ := processor.Extract(htmlContent, html.ConfigForRSS())
// 为 RSS 优化：快速、包含图片/链接、无文章检测
```

---

## API 快速参考

### 包级函数

```go
// 提取
Extract(htmlContent string) (*Result, error)
ExtractText(htmlContent string) (string, error)
ExtractFromFile(path string) (*Result, error)

// 格式转换
ExtractToMarkdown(htmlContent string) (string, error)
ExtractToJSON(htmlContent string) ([]byte, error)

// 特定元素
ExtractTitle(htmlContent string) (string, error)
ExtractImages(htmlContent string) ([]ImageInfo, error)
ExtractVideos(htmlContent string) ([]VideoInfo, error)
ExtractAudios(htmlContent string) ([]AudioInfo, error)
ExtractLinks(htmlContent string) ([]LinkInfo, error)
ExtractWithTitle(htmlContent string) (string, string, error)

// 分析
GetWordCount(htmlContent string) (int, error)
GetReadingTime(htmlContent string) (float64, error)
Summarize(htmlContent string, maxWords int) (string, error)
ExtractAndClean(htmlContent string) (string, error)

// 链接
ExtractAllLinks(htmlContent string, baseURL ...string) ([]LinkResource, error)
GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### 处理器方法

```go
// 创建
NewWithDefaults() *Processor
New(config Config) (*Processor, error)
processor.Close()

// 提取
processor.Extract(htmlContent string, config ExtractConfig) (*Result, error)
processor.ExtractWithDefaults(htmlContent string) (*Result, error)
processor.ExtractFromFile(path string, config ExtractConfig) (*Result, error)

// 批量
processor.ExtractBatch(contents []string, config ExtractConfig) ([]*Result, error)
processor.ExtractBatchFiles(paths []string, config ExtractConfig) ([]*Result, error)

// 链接
processor.ExtractAllLinks(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error)

// 监控
processor.GetStatistics() Statistics
processor.ClearCache()
```

### 配置预设

```go
DefaultExtractConfig()      ExtractConfig
ConfigForRSS()               ExtractConfig
ConfigForSummary()           ExtractConfig
ConfigForSearchIndex()       ExtractConfig
ConfigForMarkdown()          ExtractConfig
DefaultLinkExtractionConfig() LinkExtractionConfig
```

---

## 结果结构

```go
type Result struct {
    Text           string        // 清理后的文本内容
    Title          string        // 页面/文章标题
    Images         []ImageInfo   // 图片元数据
    Links          []LinkInfo    // 链接元数据
    Videos         []VideoInfo   // 视频元数据
    Audios         []AudioInfo   // 音频元数据
    WordCount      int           // 总字数
    ReadingTime    time.Duration // 预估阅读时间
    ProcessingTime time.Duration // 处理耗时
}

type ImageInfo struct {
    URL          string  // 图片 URL
    Alt          string  // Alt 文本
    Title        string  // Title 属性
    Width        string  // Width 属性
    Height       string  // Height 属性
    IsDecorative bool    // 无 alt 文本
}

type LinkInfo struct {
    URL        string  // 链接 URL
    Text       string  // 锚文本
    IsExternal bool    // 外部域名
    IsNoFollow bool    // rel="nofollow"
}
```

---

## 示例

查看 [examples/](examples) 目录获取完整可运行的代码：

| 示例 | 描述 |
|------|------|
| [01_quick_start.go](examples/01_quick_start.go) | 一行函数快速开始 |
| [02_content_extraction.go](examples/02_content_extraction.go) | 内容提取基础 |
| [03_link_extraction.go](examples/03_link_extraction.go) | 链接提取模式 |
| [04_media_extraction.go](examples/04_media_extraction.go) | 媒体（图片/视频/音频） |
| [04_advanced_features.go](examples/04_advanced_features.go) | 高级功能与兼容性 |
| [05_advanced_usage.go](examples/05_advanced_usage.go) | 批量处理与性能 |
| [06_compatibility.go](examples/06_compatibility.go) | golang.org/x/net/html 兼容性 |
| [07_convenience_api.go](examples/07_convenience_api.go) | 包级便捷 API |

---

## 兼容性

本库是 `golang.org/x/net/html` 的**直接替代品**：

```go
// 只需更改导入
- import "golang.org/x/net/html"
+ import "github.com/cybergodev/html"

// 所有现有代码都能正常工作
doc, err := html.Parse(reader)
html.Render(writer, doc)
escaped := html.EscapeString("<script>")
```

详情请参阅 [COMPATIBILITY.md](COMPATIBILITY.md)。

---

## 线程安全

`Processor` 是并发安全的：

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

## 贡献

欢迎贡献！请随时提交 Pull Request。对于重大更改，请先开启 issue 讨论。

---

## 许可证

本项目采用 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。
